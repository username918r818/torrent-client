package torrent

import (
	"context"
	"crypto/sha1"
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/username918r818/torrent-client/message"
	"github.com/username918r818/torrent-client/util"
)

type PieceState int

const (
	NotStarted PieceState = iota
	InProgress
	Downloaded
	Validated
	Saving
	Saved
	Corrupted
)

type Piece struct {
	state PieceState
	data  []byte
}

type PieceArray struct {
	pieces          []Piece
	pieceLength     int64
	lastPieceLength int64
	locks           []sync.Mutex
	listDLock       sync.Mutex                   // locks for downloaded
	downloaded      *util.List[util.Pair[int64]] // used to know ranges of downloaded but may be saved yet data
	listTLock       sync.Mutex                   // locks for toSave
	toSave          *util.List[util.Pair[int64]] // used to know ranges of downloaded but not saved yet data
	listSLock       sync.Mutex                   // locks for Saved
	Saved           *util.List[util.Pair[int64]] // used to know ranges of saved data
}

var ErrValidatedPiece = errors.New("piece already validated")

func Validate(data []byte, hash [20]byte) bool {
	return sha1.Sum(data) == hash
}

func InitPieceArray(totalBytes, pieceLength int64) (a PieceArray) {
	arrLength := totalBytes / pieceLength
	a.lastPieceLength = totalBytes % pieceLength
	if a.lastPieceLength > 0 {
		arrLength++
	} else {
		a.lastPieceLength = pieceLength
	}
	a.pieces = make([]Piece, arrLength)
	a.locks = make([]sync.Mutex, arrLength)
	a.pieceLength = pieceLength
	return
}

func UpdatePiece(pieceIndex int, a *PieceArray) ([]byte, error) {
	a.locks[pieceIndex].Lock()
	defer a.locks[pieceIndex].Unlock()
	if a.pieces[pieceIndex].state != NotStarted && a.pieces[pieceIndex].state != InProgress {
		return nil, ErrValidatedPiece
	}
	if a.pieces[pieceIndex].data == nil {
		a.pieces[pieceIndex].state = InProgress
		newLength := a.pieceLength
		if pieceIndex == len(a.pieces)-1 {
			newLength = a.lastPieceLength
		}
		a.pieces[pieceIndex].data = make([]byte, newLength)
		return a.pieces[pieceIndex].data, nil
	}

	return a.pieces[pieceIndex].data, nil
}

func StartPieceWorker(ctx context.Context, pieces *PieceArray, tf *TorrentFile, fileMap map[string]*os.File, ch message.PieceChannels) {
	for {
		select {
		case newBlock := <-ch.PeerHasDownloaded:
			// slog.Info("Piece worker: received new block")
			pieceIndex := newBlock.Offset / pieces.pieceLength
			pieceLowerBound := pieceIndex * pieces.pieceLength
			pieceUpperBound := pieceLowerBound + pieces.pieceLength
			if int(pieceIndex) == len(pieces.pieces)-1 {
				pieceUpperBound = pieceLowerBound + pieces.lastPieceLength
			}

			pieces.listDLock.Lock()
			pieces.downloaded = util.InsertRange(pieces.downloaded, newBlock.Offset, newBlock.Offset+newBlock.Length)
			checkRange := util.Contains(pieces.downloaded, pieceLowerBound, pieceUpperBound)
			pieces.listDLock.Unlock()

			ns, sai := -newBlock.Length, newBlock.Length
			validated := false
			if checkRange {
				pieces.locks[pieceIndex].Lock()
				if Validate(pieces.pieces[pieceIndex].data, tf.Pieces[pieceIndex]) {
					validated = true
					pieces.pieces[pieceIndex].state = Validated
					pieces.listTLock.Lock()
					pieces.toSave = util.InsertRange(pieces.toSave, pieceLowerBound, pieceUpperBound)
					pieces.listTLock.Unlock()
				} else {
					ns += pieceUpperBound - pieceLowerBound
				}
				sai += pieceLowerBound - pieceUpperBound
				pieces.locks[pieceIndex].Unlock()
			}
			msg := message.StatDiff{NotStarted: ns, Downloaded: sai}
			if validated {
				msg[Validated] = pieceUpperBound - pieceLowerBound
			}
			ch.PostStatsChannel <- msg

		case ready := (<-ch.FileWorkerReady):
			slog.Info("Piece worker: received new ready")
			if !ready {
				break
			}
			pieces.listTLock.Lock()
			firstRange := pieces.toSave
			if pieces.toSave != nil {
				pieces.toSave = pieces.toSave.Next
			}
			pieces.listTLock.Unlock()
			if firstRange == nil {
				msg := message.SaveRange{}
				msg.Length = -1
				ch.FileWorkerToSave <- msg
				break
			}
			totalOffset := firstRange.Value.First
			length := firstRange.Value.Second - firstRange.Value.First
			var f *os.File
			var fileStartOffset, fLength int64
			var currentOffset int64
			for _, curFile := range tf.Files {
				if currentOffset+curFile.Length > totalOffset {
					if len(curFile.Path) > 1 {
						f = fileMap[strings.Join(curFile.Path, "/")]
					} else {
						f = fileMap[curFile.Path[0]]
					}
					fileStartOffset = currentOffset
					fLength = curFile.Length
					break
				}
				currentOffset += curFile.Length
			}

			fileOffsetInFile := totalOffset - fileStartOffset
			remainingInFile := fLength - fileOffsetInFile
			if length > remainingInFile {
				length = remainingInFile
				pieces.listTLock.Lock()
				pieces.toSave = util.InsertRange(pieces.toSave, firstRange.Value.First+length, firstRange.Value.Second)
				pieces.listTLock.Unlock()
			}

			firstPiece := totalOffset / pieces.pieceLength
			lastPiece := (totalOffset + length - 1) / pieces.pieceLength

			dataToSend := make([][]byte, lastPiece+1)
			for i := firstPiece; i <= lastPiece; i++ {
				dataToSend[i] = pieces.pieces[i].data
			}

			msg := message.SaveRange{}
			msg.InfoHash = tf.InfoHash
			msg.Pieces = dataToSend
			msg.PieceLength = pieces.pieceLength
			msg.Offset = totalOffset
			msg.FileOffset = totalOffset - fileOffsetInFile
			msg.Length = length
			msg.File = f
			msg.Callback = ch.CallBack

			msgStats := message.StatDiff{Saving: length}
			ch.PostStatsChannel <- msgStats
			ch.FileWorkerToSave <- msg

		case isSaved, ok := (<-ch.FileWorkerIsSaved):
			slog.Info("Piece worker: received new saved")
			if !ok {
				break
			}
			msgStats := message.StatDiff{Saving: -isSaved.Length}
			if !isSaved.IsSaved {
				ch.PostStatsChannel <- msgStats
				pieces.listTLock.Lock()
				pieces.toSave = util.InsertRange(pieces.toSave, isSaved.Offset, isSaved.Offset+isSaved.Length)
				pieces.listTLock.Unlock()
				break
			}

			msgStats[Saved] += isSaved.Length
			ch.PostStatsChannel <- msgStats

			pieces.listSLock.Lock()
			pieces.Saved = util.InsertRange(pieces.Saved, isSaved.Offset, isSaved.Offset+isSaved.Length)
			pieces.listSLock.Unlock()

			firstPiece := isSaved.Offset / pieces.pieceLength
			lastPiece := (isSaved.Offset + isSaved.Length) / pieces.pieceLength

			for i := firstPiece; i <= lastPiece; i++ {
				if i == firstPiece || i == lastPiece {
					lw, up := i*pieces.pieceLength, (i+1)*pieces.pieceLength
					pieces.listSLock.Lock()
					checkRange := util.Contains(pieces.Saved, lw, up)
					pieces.listSLock.Unlock()
					if !checkRange {
						continue
					}
				}
				pieces.pieces[i].state = Saved
				pieces.pieces[i].data = nil
			}

		case <-ctx.Done():
			return
		}
	}

}
