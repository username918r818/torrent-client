package torrent

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/username918r818/torrent-client/file"
	"github.com/username918r818/torrent-client/message"
	"github.com/username918r818/torrent-client/util"
)

type peerState = int

const (
	PeerNotFound peerState = iota
	PeerCouldBeAdded
	PeerDead
	PeerChoking
	PeerDownloading
	PeerWaiting
)

func createBitField(pieces int) []byte {
	length := pieces / 8
	if pieces%8 != 0 {
		length++
	}
	return make([]byte, length)
}

func setPiece(piece int, bitfield []byte) {
	byteIndex, bitIndex := piece/8, piece%8
	bitfield[byteIndex] |= (1 << (7 - bitIndex))
}

func clearPiece(piece int, bitfield []byte) {
	byteIndex, bitIndex := piece/8, piece%8
	bitfield[byteIndex] &^= (1 << (7 - bitIndex))
}

func getPiece(piece int, bitfield []byte) bool {
	byteIndex, bitIndex := piece/8, piece%8
	return (bitfield[byteIndex] & (1 << (7 - bitIndex))) != 0
}

func findTask(pieceArray *PieceArray, bitfield []byte, length int, tasksPeers map[int][6]byte, peer [6]byte) (message.DownloadRange, error) {
	var msg message.DownloadRange
	msg.PieceLength = pieceArray.pieceLength
	for i, v := range pieceArray.pieces {
		if _, ok := tasksPeers[i]; !ok && (v.state == InProgress || v.state == NotStarted) && getPiece(i, bitfield) {
			switch msg.Length {
			case 0:
				msg.Offset = int64(i) * msg.PieceLength
				msg.Length = msg.PieceLength
				tasksPeers[i] = peer
			default:
				msg.Length += msg.PieceLength
				if msg.Length >= int64(length) {
					tasksPeers[i] = peer
					return msg, nil
				}

			}
		} else {
			if msg.Length > 0 {
				return msg, nil
			}
		}
	}
	if msg.Length > 0 {
		return msg, nil
	}
	return msg, errors.New("Supervisor: task not found")

}

func StartSupervisor(ctx context.Context, torrentFile TorrentFile, port int) {
	ch, traCh, peerCh, pieceCh, fileCh := message.GetChannels()
	ch.ToPeerWorkerToDownload = make(map[[6]byte]chan<- message.DownloadRange)

	trackerSession := &TrackerSession{}
	peerId := "-UT0001-" + randomDigits(12)
	copy(trackerSession.PeerId[:], peerId)

	trackerSession.TorrentFile = &torrentFile
	trackerSession.Port = port
	trackerSession.Left = torrentFile.Files[0].Length

	var wgTracker, wgFiles, wgPiece, wgPeers sync.WaitGroup
	wgTracker.Go(func() { StartWorkerTracker(ctx, trackerSession, traCh) })

	for range 1 {
		wgFiles.Go(func() { file.StartFileWorker(ctx, fileCh) })
	}

	var totalBytes int64

	fileMap, err := file.Alloc(torrentFile.Files)
	if err != nil {
		slog.ErrorContext(ctx, "Supervisor: "+err.Error())
		return
	}

	for _, v := range torrentFile.Files {
		totalBytes += v.Length

	}

	pieceFile := make(chan message.IsRangeSaved)
	pieceCh.FileWorkerIsSaved = pieceFile
	pieceCh.CallBack = pieceFile

	pieceArray := InitPieceArray(totalBytes, torrentFile.PieceLength)
	pieceCh.FileWorkerIsSaved = make(<-chan message.IsRangeSaved)

	for range 3 {
		wgPiece.Go(func() { StartPieceWorker(ctx, &pieceArray, &torrentFile, fileMap, pieceCh) })
	}

	peerState := make(map[[6]byte]peerState)
	tasksPeers := make(map[int][6]byte)
	peerTasks := make(map[[6]byte]message.DownloadRange)
	peerBitFields := make(map[[6]byte][]byte)
	var peerQueue *util.List[[6]byte]

	availablePeers := 10
	msgTest := message.PeerMessage{}
	msgTest.Id = IdReady

	peerCh.PeerMessageChannel <- msgTest
	select {
	case msg := <-ch.FromPeerWorker:
		slog.Info("Supervisor: received new message")
		switch msg.Id {
		case IdDead:
			peerState[msg.PeerId] = PeerDead
			availablePeers++
			if peerQueue != nil {
				newCh := make(chan message.DownloadRange)
				newPeerCh := peerCh
				newPeerCh.ToDownload = newCh
				ch.ToPeerWorkerToDownload[peerQueue.Value] = newCh
				wgPeers.Go(func() {
					StartPeerWorker(ctx, newPeerCh, &pieceArray, peerQueue.Value, torrentFile.InfoHash, trackerSession.PeerId)
				})
				peerState[peerQueue.Value] = PeerChoking
				availablePeers--
				peerQueue = peerQueue.Next
				if peerQueue != nil {
					peerQueue.Prev = nil
				}
			}

		case IdBitfield:
			if _, ok := peerBitFields[msg.PeerId]; !ok {
				peerBitFields[msg.PeerId] = createBitField(len(pieceArray.pieces))
			}
			copy(peerBitFields[msg.PeerId], msg.Payload)
			if peerState[msg.PeerId] == PeerWaiting {
				task, err := findTask(&pieceArray, peerBitFields[msg.PeerId], int(pieceArray.pieceLength)*3, tasksPeers, msg.PeerId)
				if err == nil {
					peerState[msg.PeerId] = PeerDownloading
					ch.ToPeerWorkerToDownload[msg.PeerId] <- task
					peerTasks[msg.PeerId] = task
				}
			}

		case IdHave:
			if _, ok := peerBitFields[msg.PeerId]; !ok {
				peerBitFields[msg.PeerId] = createBitField(len(pieceArray.pieces))
			}
			setPiece(int(msg.Payload[0]), peerBitFields[msg.PeerId])

		case IdChoke:
			peerState[msg.PeerId] = PeerChoking

		case IdUnchoke:
			peerState[msg.PeerId] = PeerWaiting
			if peerState[msg.PeerId] == PeerWaiting {
				task, err := findTask(&pieceArray, peerBitFields[msg.PeerId], int(pieceArray.pieceLength)*3, tasksPeers, msg.PeerId)
				if err == nil {
					peerState[msg.PeerId] = PeerDownloading
					ch.ToPeerWorkerToDownload[msg.PeerId] <- task
					peerTasks[msg.PeerId] = task
				}
			}

		case IdReady:
			peerState[msg.PeerId] = PeerWaiting
			if peerState[msg.PeerId] == PeerWaiting {
				task, err := findTask(&pieceArray, peerBitFields[msg.PeerId], int(pieceArray.pieceLength)*3, tasksPeers, msg.PeerId)
				if err == nil {
					peerState[msg.PeerId] = PeerDownloading
					ch.ToPeerWorkerToDownload[msg.PeerId] <- task
					peerTasks[msg.PeerId] = task
				}
			}
		}

	// case p := <-ch.GetPeers:
	// 	slog.Info("Supervisor: received peers")
	// 	for _, i := range p {
	// 		if peerState[i] == PeerNotFound {
	// 			if availablePeers > 0 {
	// 				newCh := make(chan message.DownloadRange)
	// 				newPeerCh := peerCh
	// 				newPeerCh.ToDownload = newCh
	// 				ch.ToPeerWorkerToDownload[i] = newCh
	// 				availablePeers--
	// 				wgPeers.Go(func() {
	// 					StartPeerWorker(ctx, newPeerCh, &pieceArray, i, torrentFile.InfoHash, trackerSession.PeerId)
	// 				})
	// 				peerState[i] = PeerChoking
	// 			} else {

	// 				if peerQueue == nil {
	// 					peerQueue = &util.List[[6]byte]{Prev: nil, Next: nil, Value: i}
	// 					continue
	// 				}

	// 				node := peerQueue
	// 				for node.Next != nil {
	// 					node = node.Next
	// 				}
	// 				tmp := &util.List[[6]byte]{Prev: node, Next: nil, Value: i}
	// 				node.Next = tmp
	// 			}
	// 		}
	// 	}

	// 	slog.Info("Supervisor: received peers+")

	case <-ctx.Done():
		return
	}
	slog.Info("Supervisor: loop ended")

}
