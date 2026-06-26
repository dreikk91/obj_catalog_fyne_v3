package simcommands

import "testing"

func TestBuildMCAGSM4Messages(t *testing.T) {
	t.Parallel()

	cfg := DefaultMCAGSM4Config()
	cfg.ObjectNumber = 5642
	cfg.HiddenNumber = 1234

	got, err := BuildMCAGSM4Messages(cfg)
	if err != nil {
		t.Fatalf("BuildMCAGSM4Messages error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0].Text != "&&1&2&0&1&5642&internet&091.196.053.147&3312&3311&001&0&0&0&" {
		t.Fatalf("msg1 = %q", got[0].Text)
	}
	if got[1].Text != "&&1&2&0&2&1234&internet&094.153.183.241&3312&3311&004&0&0&0&" {
		t.Fatalf("msg2 = %q", got[1].Text)
	}
	want3 := "&&1&2&0&3&5642&0&"
	if got[2].Text != want3 {
		t.Fatalf("msg3 = %q, want %q", got[2].Text, want3)
	}
}

func TestBuildMCAGSMMessages(t *testing.T) {
	t.Parallel()

	cfg := DefaultMCAGSMConfig()
	cfg.ObjectNumber = 5642
	cfg.HiddenNumber = 1234

	got, err := BuildMCAGSMMessages(cfg)
	if err != nil {
		t.Fatalf("BuildMCAGSMMessages error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Text != "&&1&2&1&5642&internet&091.196.053.147&3315&3311&240&0&0&0&" {
		t.Fatalf("msg1 = %q", got[0].Text)
	}
	if got[1].Text != "&&1&2&2&1234&internet&094.153.183.241&3315&3311&240&0&0&0&" {
		t.Fatalf("msg2 = %q", got[1].Text)
	}
}

func TestBuildMCAGSM4MessagesRequiresIP(t *testing.T) {
	t.Parallel()

	cfg := DefaultMCAGSM4Config()
	cfg.ObjectNumber = 1
	cfg.HiddenNumber = 2
	cfg.PrimaryIP = "bad"
	cfg.ReserveIP = "83.150.0.35"

	if _, err := BuildMCAGSM4Messages(cfg); err == nil {
		t.Fatal("expected invalid IP error")
	}
}
