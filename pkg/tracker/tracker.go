package tracker

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/Yash-sudo-web/vulcantorrent/pkg/bencode"
)

type RequestServer struct {
	infoHash   string
	peerID     string
	port       int
	uploaded   int
	downloaded int
	left       int
	compact    int
}

// FetchPeers detects tracker protocol and fetches peers
func FetchPeers(infoHash, trackerURL string, length int, filename string) error {
	u, err := url.Parse(trackerURL)
	if err != nil {
		return err
	}
	fmt.Printf("Fetching peers from %s...\n", u)
	switch u.Scheme {
	case "http", "https":
		return fetchPeersHTTP(infoHash, trackerURL, length, filename)
	case "udp":
		return fetchPeersUDP(infoHash, u.Host, length, filename)
	default:
		return fmt.Errorf("unsupported tracker protocol: %s", u.Scheme)
	}
}

// HTTP tracker protocol
func fetchPeersHTTP(infoHash, trackerURL string, length int, filename string) error {
	infoHashDecoded, _ := hex.DecodeString(infoHash)
	req := url.Values{}
	req.Add("info_hash", string(infoHashDecoded))
	req.Add("peer_id", "1234_69_vulcanV_0.01")
	req.Add("port", "6881")
	req.Add("uploaded", "0")
	req.Add("downloaded", "0")
	req.Add("left", strconv.Itoa(length))
	req.Add("compact", "1")

	fullURL := fmt.Sprintf("%s?%s", trackerURL, req.Encode())

	resp, err := http.Get(fullURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	decodedResponse, _, err := bencode.DecodeBencode(string(body))
	if err != nil {
		return err
	}

	responseMap := decodedResponse.(map[string]interface{})

	fmt.Println("Peers from tracker:")

	// Handle the "peers" field correctly
	if peers, ok := responseMap["peers"].([]interface{}); ok {
		for _, peer := range peers {
			peerMap := peer.(map[string]interface{})
			ip := peerMap["ip"].(string)

			// Safely handle the port field
			var port int
			switch p := peerMap["port"].(type) {
			case int64:
				port = int(p)
			case int:
				port = p
			default:
				return fmt.Errorf("unexpected port type: %T", p)
			}

			fmt.Printf("IP: %s, Port: %d\n", ip, port)
			err = DownloadFromPeer(infoHash, ip, port, filename, length)
			if err != nil {
				fmt.Printf("Failed to download from %s:%d: %v\n", ip, port, err)
			}
		}
	} else if peersString, ok := responseMap["peers"].(string); ok {
		for i := 0; i < len(peersString); i += 6 {
			ip := net.IP(peersString[i : i+4])
			port := binary.BigEndian.Uint16([]byte(peersString[i+4 : i+6]))
			fmt.Printf("IP: %s, Port: %d\n", ip, port)
			err = DownloadFromPeer(infoHash, ip.String(), int(port), filename, length)
			if err != nil {
				fmt.Printf("Failed to download from %s:%d: %v\n", ip, port, err)
			}
		}
	} else {
		return fmt.Errorf("unknown peers format")
	}

	return nil
}


// UDP tracker protocol
func fetchPeersUDP(infoHash, trackerHost string, length int, filename string) error {
	conn, err := net.Dial("udp", trackerHost)
	if err != nil {
		return err
	}
	defer conn.Close()

	transactionID := rand.Uint32()
	connectionID := uint64(0x41727101980)

	// Send connection request
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, connectionID)
	binary.Write(buf, binary.BigEndian, uint32(0)) // Action: Connect
	binary.Write(buf, binary.BigEndian, transactionID)
	_, err = conn.Write(buf.Bytes())
	if err != nil {
		return err
	}

	// Read connection response
	resp := make([]byte, 16)

	conn.SetReadDeadline(time.Now().Add(5 * time.Second)) // Set timeout
_, err = conn.Read(resp)
if err != nil {
    if os.IsTimeout(err) {
        return fmt.Errorf("timeout waiting for tracker response")
    }
    return fmt.Errorf("error reading from UDP tracker: %v", err)
}
fmt.Printf("Received response: %x\n", resp)

	
	if binary.BigEndian.Uint32(resp[4:8]) != transactionID {
		return fmt.Errorf("transaction ID mismatch")
	}
	connectionID = binary.BigEndian.Uint64(resp[8:16])

	// Send announce request
	buf.Reset()
	binary.Write(buf, binary.BigEndian, connectionID)
	binary.Write(buf, binary.BigEndian, uint32(1)) // Action: Announce
	binary.Write(buf, binary.BigEndian, transactionID)
	infoHashDecoded, _ := hex.DecodeString(infoHash)
	buf.Write(infoHashDecoded)
	buf.Write([]byte("1234_69_vulcanV_0.01")) // Peer ID
	binary.Write(buf, binary.BigEndian, uint64(0)) // Downloaded
	binary.Write(buf, binary.BigEndian, uint64(length)) // Left
	binary.Write(buf, binary.BigEndian, uint64(0)) // Uploaded
	binary.Write(buf, binary.BigEndian, uint32(2)) // Event: Started
	binary.Write(buf, binary.BigEndian, uint32(0)) // IP address
	binary.Write(buf, binary.BigEndian, uint32(rand.Uint32())) // Key
	binary.Write(buf, binary.BigEndian, int32(-1)) // Num want
	binary.Write(buf, binary.BigEndian, uint16(6881)) // Port
	_, err = conn.Write(buf.Bytes())
	if err != nil {
		return err
	}

	// Read announce response
	resp = make([]byte, 4096)
	n, err := conn.Read(resp)
	if err != nil {
		return err
	}
	if binary.BigEndian.Uint32(resp[4:8]) != transactionID {
		return fmt.Errorf("transaction ID mismatch")
	}
	if binary.BigEndian.Uint32(resp[0:4]) != 1 {
		return fmt.Errorf("invalid action in response")
	}

	// Parse peers

	fmt.Println("Peers from UDP tracker:")
	peers := resp[20:n]
	for i := 0; i < len(peers); i += 6 {
		ip := net.IP(peers[i : i+4])
		port := binary.BigEndian.Uint16(peers[i+4 : i+6])
		fmt.Printf("IP: %s, Port: %d\n", ip, port)
		err = DownloadFromPeer(infoHash, ip.String(), int(port), filename, length)
		if err != nil {
			fmt.Printf("Failed to download from %s:%d: %v\n", ip, port, err)
		}
	}

	return nil
}


func DownloadFromPeer(infoHash, peerIP string, peerPort int, outputFileName string, length int) error {
    address := fmt.Sprintf("%s:%d", peerIP, peerPort)
    if net.ParseIP(peerIP).To4() == nil {
        address = fmt.Sprintf("[%s]:%d", peerIP, peerPort)
    }

    conn, err := net.Dial("tcp", address)
    if err != nil {
        return err
    }
    defer conn.Close()

	infoHashDecoded, _ := hex.DecodeString(infoHash)
	peerID := "1234_69_vulcanV_0.01"

	// Handshake
	protocol := "BitTorrent protocol"
	handshake := make([]byte, 68)
	handshake[0] = byte(len(protocol))
	copy(handshake[1:], protocol)
	copy(handshake[28:], infoHashDecoded)
	copy(handshake[48:], peerID)

	_, err = conn.Write(handshake)
	if err != nil {
		return err
	}

	response := make([]byte, 68)
	_, err = conn.Read(response)
	if err != nil {
		return err
	}

	// Validate handshake response
	if string(response[1:20]) != protocol {
		return fmt.Errorf("invalid protocol response")
	}
	fmt.Printf("Connected to peer %s:%d\n", peerIP, peerPort)

	// Open file for writing
	file, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Simulate downloading pieces (example: downloading first piece only)
	pieceIndex := 0
	blockOffset := 0
	blockLength := 16384 // Typically 16 KB blocks in torrents

	for {
		// Request a block
		request := make([]byte, 17)
		binary.BigEndian.PutUint32(request[:4], 13) // Length of message
		request[4] = 6                              // Message ID (request)
		binary.BigEndian.PutUint32(request[5:9], uint32(pieceIndex))
		binary.BigEndian.PutUint32(request[9:13], uint32(blockOffset))
		binary.BigEndian.PutUint32(request[13:17], uint32(blockLength))

		_, err = conn.Write(request)
		if err != nil {
			return err
		}

		// Read the block data
		buffer := make([]byte, blockLength+13) // 9 bytes for header
		n, err := conn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break // End of file
			}
			return err
		}

		fmt.Print(n)

		// Validate and write the block data to file
		// if n < 13 {
		// 	return fmt.Errorf("invalid block response")
		// }

		data := buffer[:] // Skip header
		_, err = file.Write(data)
		if err != nil {
			return fmt.Errorf("failed to write data to file: %v", err)
		}

		// Update offset for next block
		blockOffset += blockLength

		// Check if the piece or file is complete
		if blockOffset >= length {
			break
		}
	}

	fmt.Printf("File successfully downloaded to %s\n", outputFileName)
	return nil
}

