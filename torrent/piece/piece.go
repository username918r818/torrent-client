package piece

import (
	"context"
	"crypto/sha1"
	"sync"

	"github.com/username918r818/torrent-client/torrent"
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
}

type rawPiece struct {
	state PieceState
}

func (r *rawPiece) GetState() PieceState {
	return r.state
}

func (r *rawPiece) SetState(s PieceState) {
	r.state = s
}

type rawPieceWithData struct {
	rawPiece
	Data []byte
}

type rawPieceWithDataAndDownloaded struct {
	rawPieceWithData
	downloaded util.List[util.Pair[int64]]
}

type rawPieceWithDataAndToSaveAndSaved struct {
	rawPieceWithData
	toSave util.List[util.Pair[int64]] // that is not sent to file workers to save
	saved  util.List[util.Pair[int64]] // that approved to be saved
}

type PieceNotStarted = rawPiece
type PieceInProgress = rawPieceWithDataAndDownloaded
type PieceDownloaded = rawPieceWithData
type PieceValidated = rawPieceWithDataAndToSaveAndSaved
type PieceSaved = rawPiece

type PieceArray struct {
	sync.RWMutex // locks on updating stats
	stats        [6]int64
	pieces       []Piece
	locks        []sync.Mutex
	listLock     sync.Mutex                  // locks for downloaded
	downloaded   util.List[util.Pair[int64]] // used to know ranges of downloaded but not saved yet data
}

func Validate(data []byte, hash [20]byte) bool {
	return sha1.Sum(data) == hash
}

func InitPieceArray(totalBytes, pieceLength int64) (a PieceArray) {
	a.stats[NotStarted] = totalBytes
	arrLength := totalBytes / pieceLength
	if totalBytes%pieceLength > 0 {
		arrLength++
	}
	a.pieces = make([]Piece, arrLength)
	a.locks = make([]sync.Mutex, arrLength)
	return
}

func StartPieceWorker(ctx context.Context, pieces *PieceArray, tf *torrent.TorrentFile) {}
