package torrent

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Yash-sudo-web/vulcantorrent/pkg/bencode"
	"github.com/Yash-sudo-web/vulcantorrent/pkg/tracker"
)

func HandleTorrentFile(filename string) error {
	fileContent, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	
	decoded, _, err := bencode.DecodeBencode(string(fileContent))
	if err != nil {
		return err
	}

	torrentMap := decoded.(map[string]interface{})
	infoDict := torrentMap["info"].(map[string]interface{})
	encodedInfo, err := bencode.EncodeBencode(infoDict)
	if err != nil {
		return err
	}

	infoHash := sha1.Sum([]byte(encodedInfo))
	infoHashHex := hex.EncodeToString(infoHash[:])
	
	trackerURL := torrentMap["announce"].(string)
	fmt.Println(infoDict)

	// Handle single-file or multi-file torrents
	var length int64
	if lenVal, ok := infoDict["length"]; ok {
		// Handle single-file torrent
		switch v := lenVal.(type) {
		case int:
			length = int64(v)
		case int64:
			length = v
		default:
			return fmt.Errorf("unexpected type for length: %T", v)
		}
	} else if files, ok := infoDict["files"].([]interface{}); ok {
		// Handle multi-file torrent
		for _, file := range files {
			fileDict := file.(map[string]interface{})
			if lenVal, ok := fileDict["length"]; ok {
				switch v := lenVal.(type) {
				case int:
					length += int64(v)
				case int64:
					length += v
				default:
					return fmt.Errorf("unexpected type for file length: %T", v)
				}
			} else {
				return fmt.Errorf("missing length in file dictionary")
			}
		}
	} else {
		return fmt.Errorf("could not determine torrent length")
	}

	fmt.Println("Connecting to tracker...")

	filename = infoDict["name"].(string)

	err = tracker.FetchPeers(infoHashHex, trackerURL, int(length), filename)
	if err != nil {
		return err
	}
	return nil
}


// Handle a magnet link
func HandleMagnetLink(magnetLink string) error {
	parsedLink := parseMagnetLink(magnetLink)
	infoHash := parsedLink["xt"]
	trackerURL := parsedLink["tr"]
	length, _ := strconv.Atoi(parsedLink["xl"])

	fmt.Println("Connecting to tracker...")
	err := tracker.FetchPeers(infoHash, trackerURL, length, "output.bin")
	if err != nil {
		return err
	}
	return nil
}

// Parse a magnet link into a map
func parseMagnetLink(magnetLink string) map[string]string {
	params := strings.Split(strings.TrimPrefix(magnetLink, "magnet:?"), "&")
	parsed := make(map[string]string)
	for _, param := range params {
		parts := strings.Split(param, "=")
		if len(parts) == 2 {
			parsed[parts[0]] = parts[1]
		}
	}
	return parsed
}
