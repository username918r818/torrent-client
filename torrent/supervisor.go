package torrent

import (
	"context"
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

func StartSupervisor(ctx context.Context, torrentFile TorrentFile, port int) {
	ch, traCh, peerCh, pieceCh, fileCh := message.GetChannels()

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

	for range 3 {
		wgPiece.Go(func() { StartPieceWorker(ctx, &pieceArray, &torrentFile, fileMap, pieceCh) })
	}

	peerState := make(map[[6]byte]peerState)
	peerBitFields := make(map[[6]byte][]byte)
	var peerQueue *util.List[[6]byte]

	availablePeers := 10

	select {
	case msg := <-ch.FromPeerWorker:
		switch msg.Id {
		case IdDead:
			peerState[msg.PeerId] = PeerDead
			availablePeers++
			if peerQueue != nil {
				wgPeers.Go(func() {
					StartPeerWorker(ctx, peerCh, &pieceArray, peerQueue.Value, torrentFile.InfoHash, trackerSession.PeerId)
				})
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

		case IdHave:
			if _, ok := peerBitFields[msg.PeerId]; !ok {
				peerBitFields[msg.PeerId] = createBitField(len(pieceArray.pieces))
			}
			setPiece(int(msg.Payload[0]), peerBitFields[msg.PeerId])
		}

	case p := <-ch.GetPeers:
		for _, i := range p {
			if peerState[i] == PeerNotFound {
				if availablePeers > 0 {
					wgPeers.Go(func() { StartPeerWorker(ctx, peerCh, &pieceArray, i, torrentFile.InfoHash, trackerSession.PeerId) })
					availablePeers--
				} else {
					if peerQueue == nil {
						peerQueue = &util.List[[6]byte]{Prev: nil, Next: nil, Value: i}
						break
					}
					node := peerQueue
					for node.
						Next != nil {
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

}
