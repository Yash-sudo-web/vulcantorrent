package bencode

import (
	"fmt"
	"strconv"
	"unicode"
)

func DecodeBencode(bencodedString string) (interface{}, int, error) {
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

	} else if bencodedString[0] == 'i' {
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
		var list []interface{}
		if len(bencodedString) == 2 {
			return make([]interface{}, 0), 2, nil
		}

		index := 1
		for index < len(bencodedString) && bencodedString[index] != 'e' {
			item, itemLength, err := DecodeBencode(bencodedString[index:])
			if err != nil {
				return nil, 0, err
			}
			list = append(list, item)
			index += itemLength
		}
		return list, index + 1, nil

	} else if bencodedString[0] == 'd' {
		var dict = make(map[string]interface{})
		index := 1
		for index < len(bencodedString) && bencodedString[index] != 'e' {
			key, keyLength, err := DecodeBencode(bencodedString[index:])
			if err != nil {
				return nil, 0, err
			}
			index += keyLength
			value, valueLength, err := DecodeBencode(bencodedString[index:])
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

func EncodeBencode(data interface{}) (string, error) {
	switch v := data.(type) {
	case string:
		return strconv.Itoa(len(v)) + ":" + v, nil
	case int:
		return "i" + strconv.Itoa(v) + "e", nil
	case []interface{}:
		encoded := "l"
		for _, item := range v {
			encodedItem, err := EncodeBencode(item)
			if err != nil {
				return "", err
			}
			encoded += encodedItem
		}
		encoded += "e"
		return encoded, nil
	case map[string]interface{}:
		encoded := "d"
		for key, value := range v {
			encodedKey, err := EncodeBencode(key)
			if err != nil {
				return "", err
			}
			encodedValue, err := EncodeBencode(value)
			if err != nil {
				return "", err
			}
			encoded += encodedKey + encodedValue
		}
		encoded += "e"
		return encoded, nil
	default:
		return "", fmt.Errorf("unsupported data type")
	}
}