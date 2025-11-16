package main

import (
	"fmt"
	"github.com/username918r818/torrent-client/util"
	"os"
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

	be, err := util.Decode(data)

	if err != nil {
		fmt.Println("Can't decode file:", err)
		return
	}

	fmt.Println(be.String())
}
