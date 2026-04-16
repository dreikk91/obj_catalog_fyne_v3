package database

import "testing"

func TestInitNamedDB_ReturnsErrorForInvalidDriver(t *testing.T) {
	db, err := InitNamedDB("missing-driver-for-test", "", "Broken")
	if err == nil {
		t.Fatal("expected error for invalid driver")
	}
	if db != nil {
		t.Fatal("expected nil db for invalid driver")
	}
}
