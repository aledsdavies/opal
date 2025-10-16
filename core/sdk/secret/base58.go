package secret

// Base58 alphabet (Bitcoin-style, no 0/O/I/l ambiguity)
const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// EncodeBase58 encodes bytes to Base58 string
// Used for compact, readable secret IDs
func EncodeBase58(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	// Convert bytes to big integer (little-endian)
	var num [8]byte
	copy(num[:], data)

	// Convert to base58
	var result []byte
	for i := 0; i < 8; i++ {
		if num[i] == 0 && i == 7 {
			continue
		}

		// Divide by 58
		var remainder byte
		for j := 0; j < 8; j++ {
			temp := int(num[j]) + int(remainder)*256
			num[j] = byte(temp / 58)
			remainder = byte(temp % 58)
		}

		result = append([]byte{base58Alphabet[remainder]}, result...)
	}

	// Handle leading zeros
	for i := 0; i < len(data); i++ {
		if data[i] != 0 {
			break
		}
		result = append([]byte{'1'}, result...)
	}

	return string(result)
}

// DecodeBase58 decodes a Base58 string to bytes
// Returns nil if the string is invalid
func DecodeBase58(s string) []byte {
	if s == "" {
		return nil
	}

	// Create reverse lookup table
	lookup := make(map[byte]byte)
	for i, c := range base58Alphabet {
		lookup[byte(c)] = byte(i)
	}

	// Decode
	var num [8]byte
	for _, c := range s {
		if c == '1' {
			// Leading zero
			continue
		}

		val, ok := lookup[byte(c)]
		if !ok {
			return nil // Invalid character
		}

		// Multiply by 58
		var carry int
		for i := 0; i < 8; i++ {
			temp := int(num[i])*58 + carry
			num[i] = byte(temp % 256)
			carry = temp / 256
		}

		if carry > 0 {
			return nil // Overflow
		}

		// Add value
		num[0] += val
	}

	return num[:]
}
