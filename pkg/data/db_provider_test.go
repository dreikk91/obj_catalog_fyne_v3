package data

import "testing"

func TestFormatDBObjectName(t *testing.T) {
	t.Parallel()

	number := int64(1003)
	title := "Офіс Регіональної служби ветконтролю на кордоні"
	got := formatDBObjectName(&number, &title)
	want := "1003 | Офіс Регіональної служби ветконтролю на кордоні"
	if got != want {
		t.Fatalf("unexpected formatted object name: got %q, want %q", got, want)
	}
}

func TestFormatDBObjectName_AlreadyPrefixed(t *testing.T) {
	t.Parallel()

	number := int64(1003)
	title := "1003 | Офіс"
	got := formatDBObjectName(&number, &title)
	if got != title {
		t.Fatalf("must keep already prefixed name, got %q", got)
	}
}
