package dialogs

func i64(v int64) *int64 {
	return &v
}

func messageTypeLabel(sc1 *int64) string {
	return adminEventTypeLabel(sc1)
}

func sc1MatchesFamily(sc1 *int64, family string) bool {
	return adminEventMatchesFamily(sc1, family)
}
