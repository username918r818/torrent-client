package torrent

import (
	"context"
	"errors"
	"fmt"
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

func findTask(pieceArray *PieceArray, bitfield []byte, length int, tasksPeers map[int][6]byte, peerTasks map[[6]byte]message.DownloadRange, peer [6]byte) (message.DownloadRange, error) {
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
					peerTasks[peer] = msg
					return msg, nil
				}

			}
		} else {
			if msg.Length > 0 {
				peerTasks[peer] = msg
				return msg, nil
			}
		}
	}
	if msg.Length > 0 {
		peerTasks[peer] = msg
		return msg, nil
	}

	return msg, errors.New("supervisor: task not found")
}

func newPeer(ctx context.Context, peerCh message.PeerChannels, peer [6]byte, pieceArray *PieceArray, infoHash, peerId [20]byte, wgPeers *sync.WaitGroup, ch *message.SupervisorChannels, peerState *map[[6]byte]peerState) {
	newCh := make(chan message.DownloadRange, 1)
	newPeerCh := peerCh
	newPeerCh.ToDownload = newCh
	ch.ToPeerWorkerToDownload[peer] = newCh
	wgPeers.Go(func() {
		StartPeerWorker(ctx, newPeerCh, pieceArray, peer, infoHash, peerId)
	})
	(*peerState)[peer] = PeerChoking
}

func deadPeer(peer [6]byte, ch *message.SupervisorChannels, peerTasks map[[6]byte]message.DownloadRange, taskPeers map[int][6]byte) {
	close(ch.ToPeerWorkerToDownload[peer])
	delete(ch.ToPeerWorkerToDownload, peer)
	task, ok := peerTasks[peer]
	if !ok {
		return
	}
	delete(peerTasks, peer)
	firstIndex := task.Offset / task.PieceLength
	lastIndex := (task.Length + task.Offset) / task.Length
	for firstIndex <= lastIndex {
		if taskPeers[int(firstIndex)] == peer {
			delete(taskPeers, int(firstIndex))
		}
		firstIndex++
	}
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

	for range 2 {
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

	for range 20 {
		wgPiece.Go(func() { StartPieceWorker(ctx, &pieceArray, &torrentFile, fileMap, pieceCh) })
	}

	peerState := make(map[[6]byte]peerState)
	tasksPeers := make(map[int][6]byte)
	peerTasks := make(map[[6]byte]message.DownloadRange)
	peerBitFields := make(map[[6]byte][]byte)
	var peerQueue *util.List[[6]byte]

	availablePeers := 5

	for {
		select {
		case msg := <-ch.FromPeerWorker:
			slog.Info(fmt.Sprintf("Supervisor: received new message with type %d", msg.Id))
			switch msg.Id {
			case IdDead:
				slog.Info("Supervisor: new dead")
				peerState[msg.PeerId] = PeerDead
				deadPeer(msg.PeerId, &ch, peerTasks, tasksPeers)
				availablePeers++
				if peerQueue != nil {

					availablePeers--
					newPeer(ctx, peerCh, peerQueue.Value, &pieceArray, torrentFile.InfoHash, trackerSession.PeerId, &wgPeers, &ch, &peerState)
					peerQueue = peerQueue.Next
					if peerQueue != nil {
						peerQueue.Prev = nil
					}
				}
				slog.Info(fmt.Sprintf("Supervisor: peers: %d", availablePeers))

			case IdBitfield:
				if _, ok := peerBitFields[msg.PeerId]; !ok {
					peerBitFields[msg.PeerId] = createBitField(len(pieceArray.pieces))
				}
				copy(peerBitFields[msg.PeerId], msg.Payload)
				if peerState[msg.PeerId] == PeerWaiting {
					task, err := findTask(&pieceArray, peerBitFields[msg.PeerId], int(pieceArray.pieceLength), tasksPeers, peerTasks, msg.PeerId)
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

				if peerState[msg.PeerId] == PeerWaiting {
					task, err := findTask(&pieceArray, peerBitFields[msg.PeerId], int(pieceArray.pieceLength), tasksPeers, peerTasks, msg.PeerId)
					if err == nil {
						peerState[msg.PeerId] = PeerDownloading
						ch.ToPeerWorkerToDownload[msg.PeerId] <- task
						peerTasks[msg.PeerId] = task
					}
				}

			case IdChoke:
				peerState[msg.PeerId] = PeerChoking

			case IdUnchoke:
				slog.Info("Supervisor: unchoke")
				peerState[msg.PeerId] = PeerWaiting
				if peerState[msg.PeerId] == PeerWaiting {
					slog.Info("Supervisor: searching for task123")

					task, err := findTask(&pieceArray, peerBitFields[msg.PeerId], int(pieceArray.pieceLength), tasksPeers, peerTasks, msg.PeerId)
					// slog.Info("Supervisor: ended search for task")

					if err == nil {
						// slog.Info("Supervisor: sent task")
						peerState[msg.PeerId] = PeerDownloading
						ch.ToPeerWorkerToDownload[msg.PeerId] <- task
						peerTasks[msg.PeerId] = task

					}
				}

				// slog.Info("Supervisor: searched for task")

			case IdReady:
				// slog.Info("Supervisor: isReady")
				peerState[msg.PeerId] = PeerWaiting
				if peerState[msg.PeerId] == PeerWaiting {
					task, err := findTask(&pieceArray, peerBitFields[msg.PeerId], int(pieceArray.pieceLength), tasksPeers, peerTasks, msg.PeerId)
					if err == nil {
						peerState[msg.PeerId] = PeerDownloading
						ch.ToPeerWorkerToDownload[msg.PeerId] <- task
						peerTasks[msg.PeerId] = task
					}
				}
			}

		case p := <-ch.GetPeers:
			// slog.Info("Supervisor: received peers")
			for _, i := range p {
				if peerState[i] == PeerNotFound {
					if availablePeers > 0 {
						availablePeers--
						newPeer(ctx, peerCh, i, &pieceArray, torrentFile.InfoHash, trackerSession.PeerId, &wgPeers, &ch, &peerState)
					} else {
						if peerQueue == nil {
							peerQueue = &util.List[[6]byte]{Prev: nil, Next: nil, Value: i}
							continue
						}
						node := peerQueue
						for node.Next != nil {
							node = node.Next
						}
						tmp := &util.List[[6]byte]{Prev: node, Next: nil, Value: i}
						node.Next = tmp
					}
				}
			}

		case <-ctx.Done():
			return
		}
		// slog.Info("Supervisor: loop ended")
	}
}
