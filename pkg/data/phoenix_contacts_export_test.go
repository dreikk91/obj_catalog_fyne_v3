package data

import (
	"database/sql"
	"strings"
	"testing"
)

func TestPhoenixAllResponsiblesQueryRemovesPanelParameter(t *testing.T) {
	query := phoenixAllResponsiblesQuery()
	if strings.Contains(query, "@p1") {
		t.Fatalf("phoenixAllResponsiblesQuery() still contains panel parameter")
	}
	if got := strings.Count(query, "LTRIM(RTRIM"); got != 2 {
		t.Fatalf("phone/email filters = %d, want 2", got)
	}
}

func TestPhoenixContactFromResponsibleRow(t *testing.T) {
	contact, ok := phoenixContactFromResponsibleRow(phoenixResponsibleRow{
		PanelID:         "L00028",
		GroupNo:         2,
		GroupName:       sql.NullString{String: "Office", Valid: true},
		ResponsibleName: sql.NullString{String: "Operator", Valid: true},
		ResponsibleAddr: sql.NullString{String: "Code", Valid: true},
		CallOrder:       sql.NullInt64{Int64: 3, Valid: true},
		ContactLabel:    sql.NullString{String: "mobile", Valid: true},
		ContactValue:    sql.NullString{String: "0500000000", Valid: true},
	})
	if !ok {
		t.Fatal("phoenixContactFromResponsibleRow() rejected a valid row")
	}
	if contact.Name != "Operator" || contact.Phone != "0500000000" || contact.Priority != 3 {
		t.Fatalf("contact = %+v", contact)
	}
	if contact.GroupNumber != 2 {
		t.Fatalf("contact group = %+v", contact)
	}
}
