package main

import (
	"fmt"
	"github.com/username918r818/torrent-client/util"
	"os"
)

func main() {
	data, err := os.ReadFile("test1.torrent")
	if err != nil {
		fmt.Println("Ошибка при чтении файла:", err)
		return
	}

	be, err := util.Decode(data)

	if err != nil {
		return
	}

	fmt.Println(be.String())
}
