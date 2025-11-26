package message

import (
	"sync"

	"github.com/username918r818/torrent-client/torrent/piece"
)

type Downloaded struct {
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
	Pieces      []piece.Piece
	PieceLength int64
	Offset      int64
	FileOffset  int64
	Length      int64
}

type DownloadRange struct {
	Pieces      []piece.Piece
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
