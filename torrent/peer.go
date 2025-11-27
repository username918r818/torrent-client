package torrent

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
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
	_, err := conn.Read(buf[:])
	if err != nil {
		return msg, err
	}
	msg.Length = binary.BigEndian.Uint32(buf[:])
	if msg.Length == 0 {
		return msg, nil
	}

	buf = make([]byte, msg.Length)
	conn.SetReadDeadline(time.Now().Add(3 * time.Minute))
	_, err = conn.Read(buf[:])
	if err != nil {
		return msg, err
	}

	msg.Id = buf[0]
	if msg.Length > 1 {
		msg.Payload = buf[1:]
	}
	return msg, nil
}

func writeMessage(conn net.Conn, msg []byte) error {
	conn.SetWriteDeadline(time.Now().Add(3 * time.Minute))
	_, err := conn.Write(msg)
	if err != nil {
		return err
	}

	return nil
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
	conn.SetReadDeadline(time.Now().Add(3 * time.Minute))
	_, err := conn.Read(buf[:])
	if err != nil {
		return err
	}

	pStrLen := buf[0]

	buf = make([]byte, pStrLen+20+20+8)
	conn.SetReadDeadline(time.Now().Add(3 * time.Minute))
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

func download(conn net.Conn, task message.DownloadRange, ch message.PeerChannels, a *PieceArray, peer [6]byte, ps *peerStatus) error {

	// slog.Info("Peer: downloading")
	curIndex := task.Offset
	requestQueue := 0

	if !ps.interested {
		err := sendInterested(conn)
		ps.interested = true
		if err != nil {
			return err
		}
	}

	for curIndex < task.Length+task.Offset || requestQueue != 0 {
		for !ps.choked && requestQueue < 5 && curIndex < task.Length+task.Offset {
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
			requestQueue++
			if err != nil {
				return err
			}
			curIndex += length
		}
		msg, err := readMessage(conn, peer)
		if err != nil {
			return err
		}
		if msg.Id == 0 && msg.Length == 0 {
			msg.Id = IdKeepAlive
		}
		switch msg.Id {
		case IdChoke:
			ps.choked = true
			return nil

		case IdPiece:
			// slog.Info("Peer: received new piece")
			index, begin, block := int(binary.BigEndian.Uint32(msg.Payload[:4])), int64(binary.BigEndian.Uint32(msg.Payload[4:8])), msg.Payload[8:]
			tmpB, err := UpdatePiece(index, a)
			if err != nil {
				slog.Error("Peer: " + err.Error())
				break
			}
			copy(tmpB[begin:], block)
			var tmpOffset, length int64 = int64(index)*int64(a.pieceLength) + int64(begin), int64(len(block))
			ch.DownloadedChannel <- message.Block{Offset: tmpOffset, Length: length}
			requestQueue--
		}
		ch.PeerMessageChannel <- msg
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
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
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

	err = sendInterested(conn)

	if err != nil {
		death(err)
		return
	}

	ps := peerStatus{true, true}

	// slog.Info("Peer worker: survived before loop")

	for {
		timer := time.NewTimer(time.Second * 1)
		select {
		case task := <-ch.ToDownload:
			// slog.Info("Peer worker: got a task")

			err = download(conn, task, ch, a, peer, &ps)
			if err != nil {
				death(err)
				timer.Stop()
				return
			}
		case <-timer.C:
			timer.Stop()
			msg, err := readMessage(conn, peer)
			if err != nil {
				death(err)
				timer.Stop()
				return
			}
			if msg.Id == IdChoke {
				ps.choked = true
			}
			if msg.Id == IdUnchoke {
				ps.choked = false
			}
			ch.PeerMessageChannel <- msg
		case <-ctx.Done():
			timer.Stop()
			return
		}
		timer.Stop()

	}

}
