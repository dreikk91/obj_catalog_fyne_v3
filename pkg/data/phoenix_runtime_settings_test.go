package data

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/config"
)

func TestPhoenixOperatorDisplayName(t *testing.T) {
	operator := PhoenixOperator{ID: 7, Name: "Іван Петренко", Login: "operator7"}
	if got := operator.DisplayName(); got != "Іван Петренко (operator7)" {
		t.Fatalf("DisplayName() = %q", got)
	}
}

func TestQuoteSQLServerIdentifierEscapesClosingBracket(t *testing.T) {
	if got := quoteSQLServerIdentifier("a]b"); got != "[a]]b]" {
		t.Fatalf("quoteSQLServerIdentifier() = %q", got)
	}
}

func TestPhoenixResponseGroupNotifyStatus(t *testing.T) {
	if got := phoenixResponseGroupNotifyStatus(17); got != "17\n" {
		t.Fatalf("phoenixResponseGroupNotifyStatus() = %q", got)
	}
}

func TestPhoenixPasswordMatchesPersonalMD5(t *testing.T) {
	const stored = "96E79218965EB72C92A549DD5A330112"
	if !phoenixPasswordMatches(stored, "111111") {
		t.Fatal("password must match the uppercase MD5 format used by Personal.Person_Psw")
	}
	if phoenixPasswordMatches(stored, "wrong-password") {
		t.Fatal("wrong password matched Personal.Person_Psw")
	}
}

func TestConfigureAlarmOperatorSelectsRolePortAndProtocolCode(t *testing.T) {
	provider := NewPhoenixDataProvider(nil, "")
	metadata := PhoenixRuntimeMetadata{ClientPort: 5051, AdminPort: 5052}

	provider.ConfigureAlarmOperator(3, "Оператор", "10.32.1.200", metadata, config.PhoenixClientRoleAdministrator)

	if provider.clientPort != 5052 {
		t.Fatalf("administrator port = %d, want 5052", provider.clientPort)
	}
	if provider.controlCenterClientCode() != "PH" {
		t.Fatalf("administrator protocol code = %q, want PH", provider.controlCenterClientCode())
	}
}
