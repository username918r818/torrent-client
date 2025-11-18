package torrent

import (
	"crypto/rand"
	"fmt"
	"github.com/username918r818/torrent-client/util"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	EventNone = iota
	EventStarted
	EventStopped
	EventCompleted
)

type TrackerSession struct {
	TorrentFile *TorrentFile
	Port        int
	PeerChannel chan<- [][6]byte
	PeerId      [20]byte
	TrackerId   string
	Interval    int

	Mutex sync.Mutex

	Event      int
	Uploaded   int
	Downloaded int
	Left       int
}

func (ts *TrackerSession) StartTracker() {
	go func() {
		peerId := "-UT0001-" + randomDigits(12)
		copy(ts.PeerId[:], peerId)
		for {
			timer := time.NewTimer(time.Duration(ts.Interval) * time.Second)
			select {
			case <-timer.C:
				ts.proceed()
			}
		}
	}()
}

func (ts *TrackerSession) proceed() {
	url := ts.TorrentFile.Announce
	infoHash := util.EncodeUrl(ts.TorrentFile.InfoHash[:])
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	url += fmt.Sprintf("%sinfo_hash=%v", sep, infoHash)
	peer_id := util.EncodeUrl(ts.PeerId[:])
	url += fmt.Sprintf("&peer_id=%v", peer_id)

	ts.Mutex.Lock()

	url += fmt.Sprintf("&port=%v", ts.Port)
	url += fmt.Sprintf("&uploaded=%v", ts.Uploaded)
	url += fmt.Sprintf("&downloaded=%v", ts.Downloaded)
	url += fmt.Sprintf("&left=%v", ts.Left)
	url += fmt.Sprintf("&compact=%v", 1)
	switch ts.Event {
	case EventStarted:
		url += fmt.Sprint("&event=started")
	case EventCompleted:
		url += fmt.Sprint("&event=completed")
	case EventStopped:
		url += fmt.Sprint("&event=stopped")
	}
	ts.Event = EventNone

	ts.Mutex.Unlock()

	resp, err := http.Get(url)
	if err != nil {
		ts.Interval = 60
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	be, err := util.Decode(body)

	if err != nil {
		ts.Interval = 60
		log.Printf("can't decode bencode: %v", err)
	}

	log.Println(be.String())

	ts.Interval = int((*be.Dict)["interval"].Int)

	peersBin := (*be.Dict)["peers"].Str
	peers := make([][6]byte, len(peersBin)/6)
	for i := 0; i < len(peers); i++ {
		copy(peers[i][:], peersBin[i*6:(i+1)*6])
	}

	ts.sendPeers(peers)

}

func (ts *TrackerSession) sendPeers(peers [][6]byte) {
	if ts.PeerChannel == nil {
		return
	}

	defer func() {
		recover()
	}()

	select {
	case ts.PeerChannel <- peers:
	default:
	}
}

func randomDigits(n int) string {
	buf := make([]byte, n)
	for i := range buf {
		b := make([]byte, 1)
		rand.Read(b)
		buf[i] = '0' + (b[0] % 10)
	}
	return string(buf)
}
