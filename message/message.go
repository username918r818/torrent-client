package message

import (
	"sync"
)

type Block struct {
	Offset int64
	Length int64
}

type Ready bool

type IsRangeSaved struct {
	IsSaved bool
	Offset  int64
	Length  int64
}

type SaveRange struct {
	InfoHash    [20]byte
	Pieces      [][]byte
	PieceLength int64
	Offset      int64
	FileOffset  int64
	Length      int64
}

type DownloadRange struct {
	Pieces      [][]byte
	locks       []sync.Mutex
	PieceLength int64
	Offset      int64
	Length      int64
}

type PeerMessage struct {
	PeerId  [6]byte
	Length  int64
	Id      byte
	Payload []byte
}

type PeerError struct {
	Error error
}

type Stats = [6]int64
