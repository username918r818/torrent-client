package torrent

import (
	"context"
	"crypto/sha1"
	"errors"
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
	sync.Mutex      // locks on updating stats
	stats           [6]int64
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

func Validate(data []byte, hash [20]byte) bool {
	return sha1.Sum(data) == hash
}

func InitPieceArray(totalBytes, pieceLength int64) (a PieceArray) {
	a.stats[NotStarted] = totalBytes
	arrLength := totalBytes / pieceLength
	a.lastPieceLength = totalBytes % pieceLength
	if a.lastPieceLength > 0 {
		arrLength++
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
		return nil, errors.New("Piece: can't update already downloaded piece")
	}
	if a.pieces[pieceIndex].state == NotStarted {
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
				if Validate(pieces.pieces[pieceIndex].data, tf.Pieces[pieceIndex]) {
					validated = true
					pieces.listTLock.Lock()
					pieces.toSave = util.InsertRange(pieces.toSave, pieceLowerBound, pieceUpperBound)
					pieces.listTLock.Unlock()
				} else {
					ns += pieceUpperBound - pieceLowerBound
				}
				sai += pieceLowerBound - pieceUpperBound
			}

			pieces.Lock()
			pieces.stats[NotStarted] += ns
			pieces.stats[Downloaded] += sai
			if validated {
				pieces.stats[Validated] += pieceUpperBound - pieceLowerBound
			}
			ch.PostStatsChannel <- pieces.stats
			pieces.Unlock()

		case ready := (<-ch.FileWorkerReady):
			if !ready {
				break
			}
			pieces.listTLock.Lock()
			firstRange := pieces.toSave
			if pieces.toSave != nil {
				pieces.toSave = pieces.toSave.Next
			}
			pieces.listTLock.Unlock()
			totalOffset := firstRange.Value.First
			length := firstRange.Value.Second - firstRange.Value.First
			var f *os.File
			var fileStartLength, fLength int64
			for _, curFile := range tf.Files {
				if fileStartLength+curFile.Length > totalOffset {
					if len(curFile.Path) > 1 {
						f = fileMap[strings.Join(curFile.Path, "/")]
					} else {
						f = fileMap[curFile.Path[0]]
					}
					fLength = curFile.Length
				}
				fileStartLength += curFile.Length
			}
			if fLength < length {
				length = fLength
				pieces.listTLock.Lock()
				pieces.toSave = util.InsertRange(pieces.toSave, firstRange.Value.First+length, firstRange.Value.Second)
				pieces.listTLock.Unlock()
			}

			firstPiece := totalOffset / pieces.pieceLength
			lastPiece := (totalOffset + length) / pieces.pieceLength

			dataToSend := make([][]byte, lastPiece+1)
			for i := firstPiece; i <= lastPiece; i++ {
				dataToSend[i] = pieces.pieces[i].data
			}

			msg := message.SaveRange{}
			msg.InfoHash = tf.InfoHash
			msg.Pieces = dataToSend
			msg.PieceLength = pieces.pieceLength
			msg.Offset = totalOffset
			msg.FileOffset = totalOffset - fileStartLength
			msg.Length = length
			msg.File = f
			msg.Callback = ch.CallBack

			ch.FileWorkerToSave <- msg

		case isSaved, ok := (<-ch.FileWorkerIsSaved):
			if !ok {
				break
			}
			if !isSaved.IsSaved {
				pieces.listTLock.Lock()
				pieces.toSave = util.InsertRange(pieces.toSave, isSaved.Offset, isSaved.Offset+isSaved.Length)
				pieces.listTLock.Unlock()
				break
			}

			pieces.listSLock.Lock()
			pieces.Saved = util.InsertRange(pieces.Saved, isSaved.Offset, isSaved.Offset+isSaved.Length)
			pieces.listSLock.Unlock()

			firstPiece := isSaved.Offset / pieces.pieceLength
			lastPiece := (isSaved.Offset + isSaved.Length) / pieces.pieceLength

			for i := firstPiece; i <= lastPiece; i++ {
				if i == firstPiece || i == lastPiece {
					lw, up := i*pieces.pieceLength, (i+1)*pieces.pieceLength
					pieces.listSLock.Lock()
					checkRange := util.Contains(pieces.downloaded, lw, up)
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
