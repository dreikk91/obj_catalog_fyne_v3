package data

import "testing"

func TestStablePhoenixID_IsDeterministicAndNamespaced(t *testing.T) {
	first := stablePhoenixID("L00028")
	second := stablePhoenixID("L00028")
	if first != second {
		t.Fatalf("stablePhoenixID must be deterministic: %d != %d", first, second)
	}
	if !IsPhoenixObjectID(first) {
		t.Fatalf("expected Phoenix namespace ID, got %d", first)
	}
	if IsCASLObjectID(first) {
		t.Fatalf("phoenix id must not overlap with CASL namespace: %d", first)
	}
}
