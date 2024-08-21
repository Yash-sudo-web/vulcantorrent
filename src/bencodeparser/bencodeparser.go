package vulcantorrent

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"unicode"
)

func decodeBencode(bencodedString string) (interface{}, int, error) {
	if len(bencodedString) == 0 {
		return nil, 0, fmt.Errorf("empty input string")
	}

	if unicode.IsDigit(rune(bencodedString[0])) {
		var firstColonIndex int
		for i := 0; i < len(bencodedString); i++ {
			if bencodedString[i] == ':' {
				firstColonIndex = i
				break
			}
		}
		lengthStr := bencodedString[:firstColonIndex]
		length, err := strconv.Atoi(lengthStr)
		if err != nil {
			return "", 0, err
		}
		return bencodedString[firstColonIndex+1 : firstColonIndex+1+length], firstColonIndex + 1 + length, nil

	} else if bencodedString[0] == 'i' && bencodedString[len(bencodedString)-1] == 'e' {
		endIndex := 1
		for endIndex < len(bencodedString) && bencodedString[endIndex] != 'e' {
			endIndex++
		}
		number, err := strconv.Atoi(bencodedString[1:endIndex])
		if err != nil {
			return nil, 0, err
		}
		return number, endIndex + 1, nil

	} else if bencodedString[0] == 'l' {
		if len(bencodedString) == 2 {
			return make([]interface{}, 0), 0, nil
		}
		var list []interface{}
		index := 1
		for index < len(bencodedString) && bencodedString[index] != 'e' {
			item, itemLength, err := decodeBencode(bencodedString[index:])
			if err != nil {
				return nil, 0, err
			}
			list = append(list, item)
			index += itemLength
		}
		return list, index + 1, nil

	} else if bencodedString[0] == 'd' {
		if len(bencodedString) == 2 {
			return make(map[string]interface{}), 0, nil
		}
		var dict = make(map[string]interface{})
		index := 1
		for index < len(bencodedString) && bencodedString[index] != 'e' {
			key, keyLength, err := decodeBencode(bencodedString[index:])
			if err != nil {
				return nil, 0, err
			}
			index += keyLength
			value, valueLength, err := decodeBencode(bencodedString[index:])
			if err != nil {
				return nil, 0, err
			}
			index += valueLength
			dict[key.(string)] = value
		}
		return dict, index + 1, nil
	} else {
		return nil, 0, fmt.Errorf("unsupported format")
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: decode <bencoded string>")
		os.Exit(1)
	}

	command := os.Args[1]

	if command == "decode" {
		bencodedValue := os.Args[2]

		decoded, _, err := decodeBencode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
