package ids

import (
	"hash/fnv"
	"strings"
)

const (
	PhoenixObjectIDNamespaceStart = 1_000_000_000
	PhoenixObjectIDNamespaceEnd   = 1_499_999_999
	PhoenixObjectIDNamespaceSize  = PhoenixObjectIDNamespaceEnd - PhoenixObjectIDNamespaceStart + 1

	CASLObjectIDNamespaceStart = 1_500_000_000
	CASLObjectIDNamespaceEnd   = 1_999_999_999
	CASLObjectIDNamespaceSize  = CASLObjectIDNamespaceEnd - CASLObjectIDNamespaceStart + 1
)

func IsCASLObjectID(id int) bool {
	return id >= CASLObjectIDNamespaceStart && id <= CASLObjectIDNamespaceEnd
}

func IsPhoenixObjectID(id int) bool {
	return id >= PhoenixObjectIDNamespaceStart && id <= PhoenixObjectIDNamespaceEnd
}

func StablePhoenixID(parts ...string) int {
	h := fnv.New32a()
	for _, part := range parts {
		_, _ = h.Write([]byte(strings.TrimSpace(part)))
		_, _ = h.Write([]byte{0})
	}

	base := int(h.Sum32() & 0x7fffffff)
	if base == 0 {
		base = 1
	}
	return PhoenixObjectIDNamespaceStart + (base % PhoenixObjectIDNamespaceSize)
}
