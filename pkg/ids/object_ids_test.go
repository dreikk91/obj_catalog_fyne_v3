package ids

import "testing"

func TestStablePhoenixIDIsDeterministicAndNamespaced(t *testing.T) {
	first := StablePhoenixID("L00028")
	second := StablePhoenixID("L00028")
	if first != second {
		t.Fatalf("StablePhoenixID must be deterministic: %d != %d", first, second)
	}
	if !IsPhoenixObjectID(first) {
		t.Fatalf("expected Phoenix namespace ID, got %d", first)
	}
	if IsCASLObjectID(first) {
		t.Fatalf("phoenix id must not overlap with CASL namespace: %d", first)
	}
}
