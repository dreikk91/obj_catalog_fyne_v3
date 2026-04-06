package data

import (
	"hash/fnv"
	"strconv"
	"strings"
)

const (
	phoenixObjectIDNamespaceStart = 1_000_000_000
	phoenixObjectIDNamespaceEnd   = 1_499_999_999
	phoenixObjectIDNamespaceSize  = phoenixObjectIDNamespaceEnd - phoenixObjectIDNamespaceStart + 1

	caslObjectIDNamespaceStart    = 1_500_000_000
	caslObjectIDNamespaceEnd      = 1_999_999_999
	caslObjectIDNamespaceSize     = caslObjectIDNamespaceEnd - caslObjectIDNamespaceStart + 1
)

func IsCASLObjectID(id int) bool {
	return id >= caslObjectIDNamespaceStart && id <= caslObjectIDNamespaceEnd
}

func IsPhoenixObjectID(id int) bool {
	return id >= phoenixObjectIDNamespaceStart && id <= phoenixObjectIDNamespaceEnd
}

func stablePhoenixID(parts ...string) int {
	h := fnv.New32a()
	for _, part := range parts {
		_, _ = h.Write([]byte(strings.TrimSpace(part)))
		_, _ = h.Write([]byte{0})
	}

	base := int(h.Sum32() & 0x7fffffff)
	if base == 0 {
		base = 1
	}
	return phoenixObjectIDNamespaceStart + (base % phoenixObjectIDNamespaceSize)
}

func stablePhoenixEventID(panelID string, eventID int64) int {
	return stablePhoenixID(strings.TrimSpace(panelID), strconv.FormatInt(eventID, 10))
}
