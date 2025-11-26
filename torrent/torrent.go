package torrent

import (
	"crypto/sha1"
	"errors"

	"github.com/username918r818/torrent-client/util"
)

// single file mode: only one file in files
type TorrentFile struct {
	Announce        string
	ReserveAnnounce []string
	Pieces          [][20]byte
	PieceLength     int64
	Files           []struct {
		Length int64
		Path   []string
	}
	InfoHash [20]byte
}

func New(data []byte) (TorrentFile, error) {
	t := TorrentFile{}
	be, err := util.Decode(data)

	if err != nil {
		return t, err
	}

	if be.Tag != util.BeDict || be.Dict == nil {
		return t, errors.New("not a dict")
	}

	hasAnnounces := false

	if _, ok := (*be.Dict)["announce"]; ok {
		t.Announce = string((*be.Dict)["announce"].Str)
		hasAnnounces = true
	}

	if _, ok := (*be.Dict)["announce-list"]; ok {
		tmp := (*be.Dict)["announce-list"]
		if tmp.Tag != util.BeList || len(tmp.List) > 0 {
			t.ReserveAnnounce = make([]string, len(tmp.List))
			for i, v := range tmp.List {
				t.ReserveAnnounce[i] = string(v.Str)
			}
			hasAnnounces = true
		}
	}

	if !hasAnnounces {
		return t, errors.New("no announces")
	}

	if info, ok := (*be.Dict)["info"]; !ok || info.Tag != util.BeDict || info.Dict == nil {
		return t, errors.New("no info")
	}

	info := *((*be.Dict)["info"].Dict)

	if _, ok := info["piece length"]; !ok {
		return t, errors.New("no piece length")
	}

	t.PieceLength = info["piece length"].Int

	if pieces, ok := info["pieces"]; ok {
		t.Pieces = make([][20]byte, len(pieces.Str)/20)
		for i := range t.Pieces {
			copy(t.Pieces[i][:], pieces.Str[i*20:(i+1)*20])
		}
	} else {
		return t, errors.New("no pieces")
	}

	name := info["name"]

	if files, ok := info["files"]; ok {
		t.Files = make([]struct {
			Length int64
			Path   []string
		}, len(files.List))
		for i, v := range files.List {
			t.Files[i].Length = (*v.Dict)["length"].Int
			t.Files[i].Path = make([]string, len((*v.Dict)["path"].List)+1)
			t.Files[i].Path[0] = string(name.Str)
			for j, w := range (*v.Dict)["path"].List {
				t.Files[i].Path[j+1] = string(w.Str)
			}

		}
	} else {
		t.Files = make([]struct {
			Length int64
			Path   []string
		}, 1)
		t.Files[0].Path = make([]string, 1)
		t.Files[0].Path[0] = string(name.Str)
		t.Files[0].Length = info["length"].Int
	}

	hBeg, hEnd, err := util.GetIndeces("info", data)
	t.InfoHash = sha1.Sum(data[hBeg:hEnd])

	return t, nil
}
