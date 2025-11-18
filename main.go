package main

import (
	"encoding/hex"
	"fmt"
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

	torrent, err := torrent.New(data)

	if err != nil {
		fmt.Println("Can't build torrent structure:", err)
		return
	}

	fmt.Println(hex.EncodeToString(torrent.InfoHash[:]))
}
