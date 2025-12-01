package torrent

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/username918r818/torrent-client/util"
)

const (
	eventNone = iota
	eventStarted
	eventStopped
	eventCompleted
)

type StatDiff struct {
	Uploaded   int64
	Downloaded int64
	Left       int64
}

type trackerSession struct {
	tf       TorrentFile
	port     int
	peerId   string
	interval int

	event      int
	uploaded   int64
	downloaded int64
	left       int64

	stats  <-chan StatDiff
	peerCh chan<- [][6]byte
}

func Init(ctx context.Context, torrentFile TorrentFile, port int, peerCh chan<- [][6]byte, statsCh <-chan StatDiff) {

	ts := trackerSession{}
	ts.tf = torrentFile
	ts.port = port
	ts.peerId = "-UT0001-" + randomDigits(12)
	ts.interval = 1
	ts.event = eventStarted

	in := make(chan [][6]byte)
	go peerHelper(ctx, in, peerCh)

	ts.peerCh = in

	go ts.start(ctx)
}

func (ts *trackerSession) start(ctx context.Context) {
	timer := time.NewTimer(time.Duration(ts.interval) * time.Second)
	for {
		select {
		case sd := <-ts.stats:
			ts.uploaded += sd.Uploaded
			ts.downloaded += sd.Downloaded
			ts.left += sd.Left
		case <-ctx.Done():
			return
		case <-timer.C:
			ts.proceed()
			timer.Reset(time.Duration(ts.interval) * time.Second)
		}
	}
}

func (ts *trackerSession) proceed() {
	url := ts.tf.Announce
	infoHash := util.EncodeUrl(ts.tf.InfoHash[:])
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	url += fmt.Sprintf("%sinfo_hash=%v", sep, infoHash)
	peer_id := util.EncodeUrl([]byte(ts.peerId))
	url += fmt.Sprintf("&peer_id=%v", peer_id)

	url += fmt.Sprintf("&port=%v", ts.port)
	url += fmt.Sprintf("&uploaded=%v", ts.uploaded)
	url += fmt.Sprintf("&downloaded=%v", ts.downloaded)
	url += fmt.Sprintf("&left=%v", ts.left)
	url += fmt.Sprintf("&compact=%v", 1)
	switch ts.event {
	case eventStarted:
		url += "&event=started"
	case eventCompleted:
		url += "&event=completed"
	case eventStopped:
		url += "&event=stopped"
	}
	ts.event = eventNone

	resp, err := http.Get(url)
	if err != nil {
		ts.interval = 60
		slog.Error("tracker: can't make request: " + err.Error())
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ts.interval = 60
		slog.Error("tracker: can't read body: " + err.Error())
		return
	}
	be, err := util.Decode(body)

	if err != nil {
		ts.interval = 60
		slog.Error("tracker: can't decode bencode: %v" + err.Error())
		return
	}

	ts.interval = int((*be.Dict)["interval"].Int)

	peersBin := (*be.Dict)["peers"].Str
	peers := make([][6]byte, len(peersBin)/6)
	for i := 0; i < len(peers); i++ {
		copy(peers[i][:], peersBin[i*6:(i+1)*6])
	}

	ts.peerCh <- peers

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

func peerHelper(ctx context.Context, in <-chan [][6]byte, out chan<- [][6]byte) {
	cur := make([][6]byte, 0)
	unique := make(map[[6]byte]struct{})
	for {
		select {
		case new := <-in:
			for _, v := range new {
				if _, ok := unique[v]; !ok {
					cur = append(cur, v)
					unique[v] = struct{}{}
				}
			}
		case out <- cur:
			cur = make([][6]byte, 0)
			unique = make(map[[6]byte]struct{})
		case <-ctx.Done():
			return
		}
	}
}
