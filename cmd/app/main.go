package main

import (
	"fmt"
	"os"

	"github.com/Yash-sudo-web/vulcantorrent/pkg/torrent"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: vulcantorrent file <filename> | magnet <magnet link>")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "file":
		filename := os.Args[2]
		err := torrent.HandleTorrentFile(filename)
		if err != nil {
			fmt.Println(err)
		}
	case "magnet":
		magnetLink := os.Args[2]
		err := torrent.HandleMagnetLink(magnetLink)
		if err != nil {
			fmt.Println(err)
		}
	default:
		fmt.Println("Unknown command")
		os.Exit(1)
	}
}
