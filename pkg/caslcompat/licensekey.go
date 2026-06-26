// Package caslcompat decodes device license keys produced by the CASL configurator.
package caslcompat

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// KeyData holds data extracted from a validated license key.
type KeyData struct {
	PPKNum int
	Key    string // numeric string representing the mobile key
}

var keyPattern = regexp.MustCompile(`^\d+-\d+-\d+-\d+-\d+-\d+$`)

const digitSet = "5169304806665065381231661576"

// ParseLicenseKey validates the license key string and extracts PPKNum and Key.
func ParseLicenseKey(raw string) (*KeyData, error) {
	if !keyPattern.MatchString(raw) {
		return nil, errors.New("license key string doesn't match format ddd-ddd-ddd-ddd-ddd-ddd")
	}
	parts := strings.Split(raw, "-")
	nums := make([]int, 6)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("part %d is not a number: %w", i+1, err)
		}
		nums[i] = n
	}
	return parseNums(nums)
}

func parseNums(nums []int) (*KeyData, error) {
	if len(nums) != 6 {
		return nil, fmt.Errorf("expected 6 elements, got %d", len(nums))
	}
	chars := make([]byte, 6)
	for i, n := range nums {
		if n < 0 || n > 255 {
			return nil, fmt.Errorf("element %d out of byte range: %d", i+1, n)
		}
		chars[i] = byte(n)
	}

	decoded := decodeLicenseString(string(chars))
	if len(decoded) < 6 {
		return nil, errors.New("decoded key too short")
	}

	// Validate checksum: first 2 chars are the checksum hex digits.
	checksumHex := fmt.Sprintf("%02x%02x", decoded[0], decoded[1])
	tail := decoded[2:]
	calc := checksumHex16(dChecksumDecStr(tail) + tail)
	if calc != checksumHex {
		return nil, errors.New("invalid license key checksum")
	}

	ppkNumHex := fmt.Sprintf("%02x%02x", decoded[2], decoded[3])
	ppkNum, err := strconv.ParseInt(ppkNumHex, 16, 32)
	if err != nil {
		return nil, fmt.Errorf("parse ppk_num: %w", err)
	}

	keyHex := fmt.Sprintf("%02x%02x", decoded[4], decoded[5])
	keyInt, err := strconv.ParseInt(keyHex, 16, 32)
	if err != nil {
		return nil, fmt.Errorf("parse key: %w", err)
	}
	if keyInt < 0 || keyInt > 65535 {
		keyInt = 0
	}

	return &KeyData{
		PPKNum: int(ppkNum),
		Key:    strconv.FormatInt(keyInt, 10),
	}, nil
}

func decodeLicenseString(s string) string {
	b := -1
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		b++
		if b == len(digitSet)-1 {
			b = 0
		}
		shift, _ := strconv.Atoi(string(digitSet[b]))
		out[i] = s[i] - byte(shift)
	}
	return string(out)
}

// checksumHex16 computes a 4-char hex checksum, matching the original JS _GetChecksum.
func checksumHex16(s string) string {
	left := 0x0056
	right := 0x00AF
	for i := 0; i < len(s); i++ {
		right += int(s[i])
		if right > 0xFF {
			right -= 0xFF
		}
		left += right
		if left > 0xFF {
			left -= 0xFF
		}
	}
	sum := (left << 8) + right
	return fmt.Sprintf("%04x", sum)
}

// dChecksumDecStr computes a decimal-string checksum, matching the original JS _GetDChecksum.
func dChecksumDecStr(s string) string {
	left := 0x0056
	right := 0x00AF
	for i := 0; i < len(s); i++ {
		right += int(s[i])
		if right > 0xFF {
			right -= 0xFF
		}
		left += right
		if left > 0xFF {
			left -= 0xFF
		}
	}
	sum := (left << 8) + right
	return strconv.Itoa(sum)
}
