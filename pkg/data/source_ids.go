package data

import (
	"strconv"
	"strings"

	"obj_catalog_fyne_v3/pkg/ids"
)

func stablePhoenixEventID(panelID string, eventID int64) int {
	return ids.StablePhoenixID(strings.TrimSpace(panelID), strconv.FormatInt(eventID, 10))
}
