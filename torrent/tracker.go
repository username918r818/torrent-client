package torrent

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/username918r818/torrent-client/message"
	"github.com/username918r818/torrent-client/util"
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
	PeerId      [20]byte
	TrackerId   string
	Interval    int

	Event      int
	Uploaded   int64
	Downloaded int64
	Left       int64
}

func StartWorkerTracker(ctx context.Context, ts *TrackerSession, ch message.TrackerChannels) {
	ts.Event = EventStarted

	tmpCounter := int64(-1)

	for {
		timer := time.NewTimer(time.Duration(ts.Interval) * time.Second)
		select {
		case <-timer.C:
			ts.proceed(ch)

		case stats := <-ch.GetStatsChannel:
			ts.Left = stats[NotStarted] + stats[Downloaded]
			ts.Downloaded = stats[Validated] + stats[Saving] + stats[Saved]
			if ts.Left == 0 {
				ts.Event = EventCompleted
			}
			if stats[Downloaded]*10000/(stats[Downloaded]+stats[NotStarted]+ts.Downloaded) > int64(tmpCounter) || true {
				tmpCounter = stats[Downloaded] * 10000 / (stats[Downloaded] + stats[NotStarted] + ts.Downloaded)
				// slog.Info(fmt.Sprintf("%v, downloaded: %v, left: %v", stats[Downloaded]*10000/(stats[Downloaded]+stats[NotStarted]), stats[Downloaded], stats[NotStarted]))
				slog.Info(fmt.Sprintf("0: %v, 1: %v, 2: %v, 3: %v, 4: %v, 5: %v", stats[0], stats[1], stats[2], stats[3], stats[4], stats[5]))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (ts *TrackerSession) proceed(ch message.TrackerChannels) {
	url := ts.TorrentFile.Announce
	infoHash := util.EncodeUrl(ts.TorrentFile.InfoHash[:])
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	url += fmt.Sprintf("%sinfo_hash=%v", sep, infoHash)
	peer_id := util.EncodeUrl(ts.PeerId[:])
	url += fmt.Sprintf("&peer_id=%v", peer_id)

	url += fmt.Sprintf("&port=%v", ts.Port)
	url += fmt.Sprintf("&uploaded=%v", ts.Uploaded)
	url += fmt.Sprintf("&downloaded=%v", ts.Downloaded)
	url += fmt.Sprintf("&left=%v", ts.Left)
	url += fmt.Sprintf("&compact=%v", 1)
	switch ts.Event {
	case EventStarted:
		url += "&event=started"
	case EventCompleted:
		url += "&event=completed"
	case EventStopped:
		url += "&event=stopped"
	}
	ts.Event = EventNone

	resp, err := http.Get(url)
	if err != nil {
		ts.Interval = 60
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ts.Interval = 60
		log.Printf("can't read body: %v", err)
	}
	be, err := util.Decode(body)

	if err != nil {
		ts.Interval = 60
		log.Printf("can't decode bencode: %v", err)
	}

	fmt.Println(be.String())
	fmt.Println(string((*be.Dict)["peers"].Str))

	ts.Interval = int((*be.Dict)["interval"].Int)

	peersBin := (*be.Dict)["peers"].Str
	peers := make([][6]byte, len(peersBin)/6)
	for i := 0; i < len(peers); i++ {
		copy(peers[i][:], peersBin[i*6:(i+1)*6])
	}

	ch.SendPeers <- peers

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
