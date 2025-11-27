package torrent

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"time"

	"github.com/username918r818/torrent-client/message"
)

const (
	IdChoke byte = iota
	IdUnchoke
	IdInterested
	IdNotInterested
	IdHave
	IdBitfield
	IdRequest
	IdPiece
	IdCancel
	IdPort
	IdKeepAlive byte = 253
	IdReady     byte = 254
	IdDead      byte = 255
)

const (
	BitTorrentPstr = "BitTorrent protocol"
)

const (
	BlockSize = 1 << 14
)

type peerStatus struct {
	choked     bool
	interested bool
}

func readMessage(conn net.Conn, peerId [6]byte) (message.PeerMessage, error) {
	msg := message.PeerMessage{}
	msg.PeerId = peerId
	conn.SetReadDeadline(time.Now().Add(3 * time.Minute))
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return msg, err
	}
	msg.Length = binary.BigEndian.Uint32(buf[:])
	if msg.Length == 0 {
		msg.Id = IdKeepAlive
		return msg, nil
	}

	buf = make([]byte, msg.Length)
	conn.SetReadDeadline(time.Now().Add(3 * time.Minute))
	if _, err := io.ReadFull(conn, buf); err != nil {
		return msg, err
	}

	msg.Id = buf[0]
	if msg.Length > 1 {
		msg.Payload = buf[1:]
	}
	return msg, nil
}

func infiniteReadingMessage(conn net.Conn, peerId [6]byte, toWriter chan<- byte, toSup chan<- message.PeerMessage, toPiece chan<- message.Block, a *PieceArray) {
	for {
		msg, err := readMessage(conn, peerId)
		if err != nil {
			slog.Error("peer reader: " + err.Error())
			dm := message.PeerMessage{PeerId: peerId, Id: IdDead}
			toSup <- dm
			toWriter <- IdDead
			return
		}
		if msg.Id == 0 && msg.Length == 0 {
			msg.Id = IdKeepAlive
		}
		toSup <- msg
		toWriter <- msg.Id
		if msg.Id == IdPiece {
			index, begin, block := int(binary.BigEndian.Uint32(msg.Payload[:4])), int64(binary.BigEndian.Uint32(msg.Payload[4:8])), msg.Payload[8:]
			tmpB, err := UpdatePiece(index, a)
			if err != nil {
				slog.Error("Peer: " + err.Error())
				toWriter <- IdDead
				return
			}
			copy(tmpB[begin:], block)
			var tmpOffset, length int64 = int64(index)*int64(a.pieceLength) + int64(begin), int64(len(block))
			toPiece <- message.Block{Offset: tmpOffset, Length: length}
		}
	}
}

func writeMessage(conn net.Conn, msg []byte) error {
	conn.SetWriteDeadline(time.Now().Add(3 * time.Minute))
	_, err := conn.Write(msg)
	if err != nil {
		return err
	}

	return nil
}

func sendBitField(conn net.Conn, length int) error {
	msg := make([]byte, 5+length)
	binary.BigEndian.PutUint32(msg[0:4], uint32(length+1))
	msg[4] = IdBitfield
	return writeMessage(conn, msg)
}

func sendInterested(conn net.Conn) error {
	msg := make([]byte, 5)
	binary.BigEndian.PutUint32(msg[0:4], 1)
	msg[4] = IdInterested
	return writeMessage(conn, msg)
}

func sendRequest(conn net.Conn, index, begin, length uint32) error {
	msg := make([]byte, 17)
	binary.BigEndian.PutUint32(msg[0:4], 13)
	msg[4] = IdRequest
	binary.BigEndian.PutUint32(msg[5:9], index)
	binary.BigEndian.PutUint32(msg[9:13], begin)
	binary.BigEndian.PutUint32(msg[13:17], length)
	return writeMessage(conn, msg)
}

func handshakeRead(conn net.Conn, infoHash [20]byte) error {
	buf := make([]byte, 1)
	conn.SetReadDeadline(time.Now().Add(50 * time.Second))
	_, err := conn.Read(buf[:])
	if err != nil {
		return err
	}

	pStrLen := buf[0]

	buf = make([]byte, pStrLen+20+20+8)
	conn.SetReadDeadline(time.Now().Add(50 * time.Second))
	_, err = conn.Read(buf[:])
	if err != nil {
		return err
	}

	if BitTorrentPstr != string(buf[:pStrLen]) {
		return fmt.Errorf("peer: wrong protocol")
	}

	if !bytes.Equal(infoHash[:], buf[pStrLen+8:pStrLen+8+20]) {
		return fmt.Errorf("peer: wrong hash_info")
	}
	return nil
}

func handshakeWrite(conn net.Conn, pstr string, infoHash [20]byte, peerId [20]byte) error {
	pstrB := []byte(pstr)
	var reserved [8]byte
	msg := make([]byte, 49+len(pstrB))
	msg[0] = byte(len(pstrB))
	copy(msg[1:], pstrB)
	copy(msg[1+len(pstrB):], reserved[:])
	copy(msg[9+len(pstrB):], infoHash[:])
	copy(msg[29+len(pstrB):], peerId[:])
	return writeMessage(conn, msg)
}

func download(conn net.Conn, task message.DownloadRange, ch message.PeerChannels, a *PieceArray, peer [6]byte, ps *peerStatus, fromReader <-chan byte) error {
	// slog.Info("Peer: downloading")
	curIndex := task.Offset

	requestCounter := 0

	if !ps.interested {
		err := sendInterested(conn)
		ps.interested = true
		if err != nil {
			return err
		}
	}

	for curIndex < task.Length+task.Offset || requestCounter > 0 {
		for !ps.choked && curIndex < task.Length+task.Offset && requestCounter < 5 {
			index := curIndex / a.pieceLength
			begin := curIndex % a.pieceLength
			length := int64(BlockSize)
			if begin+length > a.pieceLength {
				length = a.pieceLength - begin
			}
			if curIndex+length > task.Offset+task.Length {
				length = task.Offset + task.Length - curIndex
			}
			err := sendRequest(conn, uint32(index), uint32(begin), uint32(length))
			if err != nil {
				return err
			}
			requestCounter++
			curIndex += length
		}
		exitLoop := false
		for !exitLoop {
			select {
			case msgId := <-fromReader:
				switch {
				case msgId == IdChoke:
					ps.choked = true
					return nil
				case msgId == IdUnchoke:
					ps.choked = false
				case msgId == IdPiece:
					requestCounter--
				case msgId == IdDead:
					return errors.New("peer: received dead signal from reader")
				}

			default:
				exitLoop = true
			}
		}
	}

	msg := message.PeerMessage{}
	msg.Id = IdReady
	msg.PeerId = peer
	ch.PeerMessageChannel <- msg

	return nil
}

func StartPeerWorker(ctx context.Context, ch message.PeerChannels, a *PieceArray, peer [6]byte, infoHash [20]byte, peerId [20]byte) {

	ip := fmt.Sprintf("%d.%d.%d.%d", peer[0], peer[1], peer[2], peer[3])
	port := int64(peer[4])<<8 | int64(peer[5])
	addr := ip + ":" + strconv.FormatInt(port, 10)
	conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
	death := func(err error) {
		slog.Info("Peer: " + err.Error())
		msg := message.PeerMessage{}
		msg.PeerId = peer
		msg.Id = IdDead
		ch.PeerMessageChannel <- msg
	}

	if err != nil {
		death(err)
		return
	}

	err = handshakeWrite(conn, BitTorrentPstr, infoHash, peerId)

	if err != nil {
		death(err)
		return
	}

	err = handshakeRead(conn, infoHash)

	if err != nil {
		death(err)
		return
	}

	msg, err := readMessage(conn, peer)

	if err != nil {
		death(err)
		return
	}

	ch.PeerMessageChannel <- msg

	if msg.Id != IdBitfield {
		death(errors.New("peer: not bitfield after handshake"))
		return
	}

	// err = sendBitField(conn, len(a.pieces))

	// if err != nil {
	// 	death(err)
	// 	return
	// }

	fromReader := make(chan byte)

	go infiniteReadingMessage(conn, peer, fromReader, ch.PeerMessageChannel, ch.DownloadedChannel, a)

	time.Sleep(5 * time.Second)

	err = sendInterested(conn)

	if err != nil {
		death(err)
		return
	}

	ps := peerStatus{true, true}

	// slog.Info("Peer worker: survived before loop")

	for {
		timer := time.NewTimer(time.Millisecond * 5000)
		select {
		case task := <-ch.ToDownload:
			// slog.Info("Peer worker: got a task")

			err = download(conn, task, ch, a, peer, &ps, fromReader)
			if err != nil {
				death(err)
				timer.Stop()
				return
			}
		case <-timer.C:
			timer.Stop()
			var keepAlive [4]byte
			err := writeMessage(conn, keepAlive[:])
			if err != nil {
				death(err)
				timer.Stop()
				return
			}
		case byteId := <-fromReader:
			switch byteId {
			case IdChoke:
				ps.choked = true

			case IdUnchoke:
				ps.choked = false

			case IdDead:
				death(errors.New("peer: reader died"))
			}

		case <-ctx.Done():
			timer.Stop()
			return
		}
		timer.Stop()

	}

}
