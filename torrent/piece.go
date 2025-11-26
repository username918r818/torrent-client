package torrent

import (
	"context"
	"crypto/sha1"
	"errors"
	"os"
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

type Piece interface {
	GetState() PieceState
	SetState(PieceState)
	GetData() []byte
	SetData(data []byte)
}

type rawPiece struct {
	state PieceState
	data  []byte
}

func (r *rawPiece) GetState() PieceState {
	return r.state
}

func (r *rawPiece) SetState(s PieceState) {
	r.state = s
}

func (r *rawPiece) GetData() []byte {
	return r.data
}

func (r *rawPiece) SetData(data []byte) {
	r.data = data
}

type PieceArray struct {
	sync.Mutex      // locks on updating stats
	stats           [6]int64
	pieces          []Piece
	pieceLength     int64
	lastPieceLength int64
	locks           []sync.Mutex
	listDLock       sync.Mutex                  // locks for downloaded
	downloaded      util.List[util.Pair[int64]] // used to know ranges of downloaded but not saved yet data
	listSLock       sync.Mutex                  // locks for toSaved
	toSave          util.List[util.Pair[int64]] // used to know ranges of downloaded but not saved yet data
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
	if a.pieces[pieceIndex].GetState() != NotStarted && a.pieces[pieceIndex].GetState() != InProgress {
		return nil, errors.New("Piece: can't update already downloaded piece")
	}
	if a.pieces[pieceIndex].GetState() == NotStarted {
		a.pieces[pieceIndex].SetState(InProgress)
		newLength := a.pieceLength
		if pieceIndex == len(a.pieces)-1 {
			newLength = a.lastPieceLength
		}
		a.pieces[pieceIndex].SetData(make([]byte, newLength))
		return a.pieces[pieceIndex].GetData(), nil
	}

	return a.pieces[pieceIndex].GetData(), nil
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
			pieces.downloaded = *util.InsertRange(&pieces.downloaded, newBlock.Offset, newBlock.Offset+newBlock.Length)
			checkRange := util.Contains(&pieces.downloaded, pieceLowerBound, pieceUpperBound)
			pieces.listDLock.Unlock()

			ns, sai := -newBlock.Length, newBlock.Length
			validated := false
			if checkRange {
				if Validate(pieces.pieces[pieceIndex].GetData(), tf.Pieces[pieceIndex]) {
					validated = true
					pieces.listSLock.Lock()
					pieces.toSave = *util.InsertRange(&pieces.toSave, pieceLowerBound, pieceUpperBound)
					pieces.listSLock.Unlock()
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

		case ready := (<-ch.FileWorkerReady): // TODO
			_ = ready

		case isSaved, ok := (<-ch.FileWorkerIsSaved): // TODO
			_, _ = isSaved, ok

		case <-ctx.Done():
			return
		}
	}

}
