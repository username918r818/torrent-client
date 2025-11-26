package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/username918r818/torrent-client/torrent"
)

func main() {
	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) != 1 {
		fmt.Println("Need only one arg (torrent-file location)")
		return
	}

	data, err := os.ReadFile(argsWithoutProg[0])
	if err != nil {
		fmt.Println("Can't read file:", err)
		return
	}

	torrentFile, err := torrent.New(data)

	if err != nil {
		fmt.Println("Can't build torrent structure:", err)
		return
	}

	// fmt.Println(hex.EncodeToString(torrent.InfoHash[:]))

	tmp := torrent.TrackerSession{}

	tmp.TorrentFile = &torrentFile
	tmp.Port = 1488
	tmp.Event = torrent.EventStarted
	tmp.Left = int(torrentFile.Files[0].Length)

	peerCh := make(chan [][6]byte, 10)
	tmp.PeerChannel = peerCh

	go func() {
		for peers := range peerCh {
			log.Println("Received peers:", len(peers))
			for _, p := range peers {
				ip := fmt.Sprintf("%d.%d.%d.%d", p[0], p[1], p[2], p[3])
				port := int(p[4])<<8 | int(p[5])
				log.Printf("%s:%d\n", ip, port)
			}
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	torrent.StartWorkerTracker(ctx, &tmp)
	select {}
}
