package piece

import (
	"crypto/sha1"
	"github.com/username918r818/torrent-client/util"
	"sync"
)

type PieceState int

const (
	NotStarted PieceState = iota
	InProgress
	Downloaded
	Validated
	Saved
	Deleted
)

type Piece interface {
	GetState() PieceState
	SetState(PieceState)
}

type rawPiece struct {
	sync.RWMutex
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
	Hash [20]byte
	Data []byte
}

type rawPieceWithDataAndBlockList struct {
	rawPieceWithData
	downloaded util.List[util.Pair[int64]]
}

type PieceNotStarted = rawPiece
type PieceInProgress = rawPieceWithDataAndBlockList
type PieceDownloaded = rawPieceWithData
type PieceValidated = rawPieceWithData
type PieceSaved = rawPiece

type PieceArray struct {
	sync.RWMutex
	length [5]int64
	pieces []Piece
}

func Validate(data []byte, hash [20]byte) bool {
	return sha1.Sum(data) == hash
}

func InitPieceArray(totalBytes, pieceLength int64) (a PieceArray) {
	a.length[NotStarted] = totalBytes
	arrLength := totalBytes / pieceLength
	if totalBytes%pieceLength > 0 {
		arrLength++
	}
	a.pieces = make([]Piece, arrLength)
	return
}
