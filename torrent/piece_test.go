package torrent_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/username918r818/torrent-client/torrent"
)

func TestPieceValidation(t *testing.T) {
	t.Run("successful validating", func(t *testing.T) {
		data := []byte("This page intentionally left blank.")

		hexStr := "af 06 49 23 bb f2 30 15 96 aa c4 c2 73 ba 32 17 8e bc 4a 96"
		hexStr = strings.ReplaceAll(hexStr, " ", "")
		hashS, err := hex.DecodeString(hexStr)
		if err != nil {
			t.Fatal(err)
		}

		var hash [20]byte
		copy(hash[:], hashS)

		if !torrent.Validate(data, hash) {
			t.Errorf("expected validation to succeed, but it failed")
		}
	})

	t.Run("failed validating", func(t *testing.T) {
		data := []byte("This page intentionally right blank.")

		hexStr := "af 06 49 23 bb f2 30 15 96 aa c4 c2 73 ba 32 17 8e bc 4a 96"
		hexStr = strings.ReplaceAll(hexStr, " ", "")
		hashS, err := hex.DecodeString(hexStr)
		if err != nil {
			t.Fatal(err)
		}

		var hash [20]byte
		copy(hash[:], hashS)

		if torrent.Validate(data, hash) {
			t.Errorf("expected validation to fail, but it succeeded")
		}
	})
}
