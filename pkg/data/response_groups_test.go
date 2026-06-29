package data

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func TestPhoenixResponseGroupStatus(t *testing.T) {
	tests := []struct {
		id   int64
		want contracts.ResponseGroupStatus
	}{
		{id: 1, want: contracts.ResponseGroupStatusFree},
		{id: 2, want: contracts.ResponseGroupStatusDispatched},
		{id: 3, want: contracts.ResponseGroupStatusArrived},
		{id: 99, want: contracts.ResponseGroupStatusUnknown},
	}
	for _, test := range tests {
		if got := phoenixResponseGroupStatus(test.id); got != test.want {
			t.Fatalf("phoenixResponseGroupStatus(%d) = %q, want %q", test.id, got, test.want)
		}
	}
}

func TestCASLResponseGroupStatus(t *testing.T) {
	tests := map[string]contracts.ResponseGroupStatus{
		"":                        contracts.ResponseGroupStatusFree,
		"GRD_OBJ_ASS_MGR":         contracts.ResponseGroupStatusDispatched,
		"GRD_OBJ_MGR_ARRIVE":      contracts.ResponseGroupStatusArrived,
		"GRD_OBJ_MGR_CANCEL":      contracts.ResponseGroupStatusFree,
		"UNRECOGNIZED_MGR_ACTION": contracts.ResponseGroupStatusUnknown,
	}
	for action, want := range tests {
		if got := caslResponseGroupStatus(action); got != want {
			t.Fatalf("caslResponseGroupStatus(%q) = %q, want %q", action, got, want)
		}
	}
}
