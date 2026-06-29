//go:build qt

package qtui

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func TestFilterResponseGroupsBySourceStatusAndSearch(t *testing.T) {
	groups := []contracts.FrontendResponseGroup{
		{
			ID: "1", Name: "Захід", Callsign: "Сокіл",
			Source: contracts.FrontendSourceCASL, Status: contracts.ResponseGroupStatusDispatched,
			ObjectNumber: "1007",
		},
		{
			ID: "2", Name: "Північ",
			Source: contracts.FrontendSourcePhoenix, Status: contracts.ResponseGroupStatusFree,
		},
	}

	got := filterResponseGroups(groups, "сокіл", contracts.FrontendSourceCASL.DisplayName(), "Направлені")
	if len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("filtered groups = %+v, want CASL group 1", got)
	}
}

func TestFilterResponseGroupsOrdersActiveBeforeFree(t *testing.T) {
	groups := []contracts.FrontendResponseGroup{
		{ID: "free", Name: "А", Status: contracts.ResponseGroupStatusFree},
		{ID: "arrived", Name: "Б", Status: contracts.ResponseGroupStatusArrived},
		{ID: "sent", Name: "В", Status: contracts.ResponseGroupStatusDispatched},
	}
	got := filterResponseGroups(groups, "", responseGroupsAllSources, responseGroupsAllStates)
	if len(got) != 3 || got[0].ID != "sent" || got[1].ID != "arrived" || got[2].ID != "free" {
		t.Fatalf("unexpected response group order: %+v", got)
	}
}
