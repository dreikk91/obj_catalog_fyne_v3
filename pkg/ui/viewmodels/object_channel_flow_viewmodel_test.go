package viewmodels

import "testing"

func TestObjectChannelFlowViewModel_ResolveChange(t *testing.T) {
	vm := NewObjectChannelFlowViewModel()

	out := vm.ResolveChange(
		"5 - GPRS",
		"PPK-100",
		map[string]int64{
			"1 - Автододзвон": 1,
			"5 - GPRS":        5,
		},
		func(label string) int64 {
			if label == "PPK-100" {
				return 100
			}
			return 0
		},
	)

	if out.ChannelCode != 5 {
		t.Fatalf("unexpected channel code: %d", out.ChannelCode)
	}
	if out.PreferredPPKID != 100 {
		t.Fatalf("unexpected preferred ppk id: %d", out.PreferredPPKID)
	}
}

func TestObjectChannelFlowViewModel_ResolveChange_Fallbacks(t *testing.T) {
	vm := NewObjectChannelFlowViewModel()

	out := vm.ResolveChange(
		"unknown",
		"PPK-200",
		map[string]int64{
			"1 - Автододзвон": 1,
		},
		nil,
	)

	if out.ChannelCode != 1 {
		t.Fatalf("expected default channel code, got: %d", out.ChannelCode)
	}
	if out.PreferredPPKID != 0 {
		t.Fatalf("expected zero preferred ppk id without lookup, got: %d", out.PreferredPPKID)
	}
}
