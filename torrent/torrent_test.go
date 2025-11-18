package torrent_test

import (
	"encoding/hex"
	"testing"

	"github.com/username918r818/torrent-client/torrent"
)

func TestTorrentSingleFile(t *testing.T) {
	data := []byte(
		"d8:announce19:http://tracker1.com4:infod12:piece lengthi16384e6:pieces40:ABCDEFGHIJKLMNOPQRSTABCDEFGHIJKLMNOPQRST4:name8:file.txt6:lengthi1000eee",
	)

	tr, err := torrent.New(data)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if tr.Announce != "http://tracker1.com" {
		t.Fatalf("wrong announce: %q", tr.Announce)
	}

	if len(tr.ReserveAnnounce) != 0 {
		t.Fatalf("expected no reserve announces, got=%v", tr.ReserveAnnounce)
	}

	if tr.PieceLength != 16384 {
		t.Fatalf("wrong piece length: %v", tr.PieceLength)
	}

	if len(tr.Pieces) != 2 {
		t.Fatalf("wrong piece count: %v", len(tr.Pieces))
	}

	if tr.Pieces[0] != [20]byte{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T'} {
		t.Fatalf("unexpected first piece content")
	}

	if len(tr.Files) != 1 {
		t.Fatalf("should be single-file mode")
	}

	expectedHash := "dab44c142c1450e6ff20a0eec3e6a983b626cdec"

	if hex.EncodeToString(tr.InfoHash[:]) != expectedHash {
		t.Fatalf("wrong hash\ngot:\n%v\nexpected:\n%v", hex.EncodeToString(tr.InfoHash[:]), expectedHash)
	}
}

func TestTorrentMultiFile(t *testing.T) {
	data := []byte(
		"d8:announce19:http://tracker1.com4:infod12:piece lengthi32768e6:pieces40:ABCDEFGHIJKLMNOPQRSTABCDEFGHIJKLMN" +
			"OPQRST4:name4:root5:filesld6:lengthi111e4:pathl9:fileA.txteed6:lengthi222e4:pathl9:fileB.txteeeee",
	)

	tr, err := torrent.New(data)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if tr.Announce != "http://tracker1.com" {
		t.Fatalf("wrong announce: %q", tr.Announce)
	}

	if tr.PieceLength != 32768 {
		t.Fatalf("wrong piece length: %v", tr.PieceLength)
	}

	if len(tr.Pieces) != 2 {
		t.Fatalf("wrong piece count: %v", len(tr.Pieces))
	}

	if len(tr.Files) != 2 {
		t.Fatalf("expected 2 files, got %v", len(tr.Files))
	}

	expectedHash := "a9e1ca2b74100277ec48a4d4c7264e0c17e35ff9"

	if hex.EncodeToString(tr.InfoHash[:]) != expectedHash {
		t.Fatalf("wrong hash\ngot:\n%v\nexpected:\n%v", hex.EncodeToString(tr.InfoHash[:]), expectedHash)
	}
}
