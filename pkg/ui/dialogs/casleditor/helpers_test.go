package casleditor

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func TestNormalizeCASLEditorSIM(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "formatted", input: "+38 (050) 123-45-67", want: "+38 (050) 123-45-67"},
		{name: "digits 12", input: "380501234567", want: "+38 (050) 123-45-67"},
		{name: "digits 10", input: "0501234567", want: "+38 (050) 123-45-67"},
		{name: "digits 11", input: "80501234567", want: "+38 (050) 123-45-67"},
		{name: "messy international", input: " +380 (50) 123 45 67 ", want: "+38 (050) 123-45-67"},
		{name: "messy local", input: "050 123 45 67", want: "+38 (050) 123-45-67"},
		{name: "empty", input: "", want: ""},
		{name: "partial", input: "+38 (050) 123-__", wantErr: true},
		{name: "invalid", input: "12345", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeCASLEditorSIM(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NormalizeCASLEditorSIM(%q) expected error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeCASLEditorSIM(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeCASLEditorSIM(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeCASLEditorUserPhone(t *testing.T) {
	t.Parallel()

	got, err := NormalizeCASLEditorUserPhone("0501234567")
	if err != nil {
		t.Fatalf("NormalizeCASLEditorUserPhone() unexpected error: %v", err)
	}
	if got != "+38 (050) 123-45-67" {
		t.Fatalf("NormalizeCASLEditorUserPhone() = %q", got)
	}

	got, err = NormalizeCASLEditorUserPhone("")
	if err != nil {
		t.Fatalf("NormalizeCASLEditorUserPhone(empty) unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("NormalizeCASLEditorUserPhone(empty) = %q", got)
	}

	if _, err := NormalizeCASLEditorUserPhone("12345"); err == nil {
		t.Fatal("NormalizeCASLEditorUserPhone() expected error for invalid input")
	}

	got, err = NormalizeCASLEditorUserPhone(" +380 (67) 111 22 33 ")
	if err != nil {
		t.Fatalf("NormalizeCASLEditorUserPhone(messy) unexpected error: %v", err)
	}
	if got != "+38 (067) 111-22-33" {
		t.Fatalf("NormalizeCASLEditorUserPhone(messy) = %q", got)
	}
}

func TestTryFormatCASLEditorUserPhone(t *testing.T) {
	t.Parallel()

	if got, ok := TryFormatCASLEditorUserPhone("0501234567"); !ok || got != "+38 (050) 123-45-67" {
		t.Fatalf("TryFormatCASLEditorUserPhone() = %q, %v", got, ok)
	}
	if got, ok := TryFormatCASLEditorUserPhone("05012"); ok || got != "" {
		t.Fatalf("TryFormatCASLEditorUserPhone(short) = %q, %v", got, ok)
	}
}

func TestFormatCASLEditorPhoneProgressively(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "0", want: "+38 (0"},
		{input: "050", want: "+38 (050)"},
		{input: "0503", want: "+38 (050) 3"},
		{input: "0503299204", want: "+38 (050) 329-92-04"},
		{input: "+380503299204", want: "+38 (050) 329-92-04"},
	}

	for _, tt := range tests {
		if got := FormatCASLEditorPhoneProgressively(tt.input); got != tt.want {
			t.Fatalf("FormatCASLEditorPhoneProgressively(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNextCASLLineNumber(t *testing.T) {
	t.Parallel()

	lines := []contracts.CASLDeviceLineDetails{
		{LineNumber: 1},
		{LineNumber: 2},
		{LineNumber: 4},
	}
	if got := NextCASLLineNumber(lines); got != 3 {
		t.Fatalf("NextCASLLineNumber() = %d, want 3", got)
	}
}

func TestNormalizeCASLEditorLicenceForSave(t *testing.T) {
	t.Parallel()

	got, err := NormalizeCASLEditorLicenceForSave("123-456-789-012-345-678")
	if err != nil {
		t.Fatalf("NormalizeCASLEditorLicenceForSave() unexpected error: %v", err)
	}
	if got != "123;456;789;012;345;678" {
		t.Fatalf("NormalizeCASLEditorLicenceForSave() = %q", got)
	}

	if _, err := NormalizeCASLEditorLicenceForSave("123-456"); err == nil {
		t.Fatal("NormalizeCASLEditorLicenceForSave() expected error for short key")
	}
}

func TestValidateCASLEditorDateField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "empty", input: "", wantErr: false},
		{name: "iso", input: "2026-04-16", wantErr: false},
		{name: "ua", input: "16.04.2026", wantErr: false},
		{name: "ua short", input: "6.4.2026", wantErr: false},
		{name: "invalid", input: "parsing time \"\"", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCASLEditorDateField(tt.input)
			if tt.wantErr && err == nil {
				t.Fatalf("ValidateCASLEditorDateField(%q) expected error", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("ValidateCASLEditorDateField(%q) unexpected error: %v", tt.input, err)
			}
		})
	}
}
