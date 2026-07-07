//go:build qt

package qtui

import (
	"hash"
	"strconv"
)

func writeHashString(h hash.Hash64, value string) {
	_, _ = h.Write([]byte(value))
	_, _ = h.Write([]byte{0})
}

func writeHashInt(h hash.Hash64, value int) {
	writeHashString(h, strconv.Itoa(value))
}

func writeHashInt64(h hash.Hash64, value int64) {
	writeHashString(h, strconv.FormatInt(value, 10))
}

func writeHashBool(h hash.Hash64, value bool) {
	if value {
		_, _ = h.Write([]byte{1})
		return
	}
	_, _ = h.Write([]byte{2})
}
