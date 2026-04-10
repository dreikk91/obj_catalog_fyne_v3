package data

import "testing"

func TestDecodeCASLSystemCode_TableDriven(t *testing.T) {
	tests := []struct {
		name string
		b1   byte
		b2   byte
		want caslDecodedEventCode
		ok   bool
	}{
		{
			name: "ban time",
			b1:   0x00,
			b2:   0xB3,
			want: caslDecodedEventCode{MessageKey: "BAN_TIME"},
			ok:   true,
		},
		{
			name: "ppk conn ok",
			b1:   0x00,
			b2:   0x60,
			want: caslDecodedEventCode{MessageKey: "PPK_CONN_OK"},
			ok:   true,
		},
		{
			name: "oo no ping",
			b1:   0x01,
			b2:   0x62,
			want: caslDecodedEventCode{MessageKey: "OO_NO_PING"},
			ok:   true,
		},
		{
			name: "unknown",
			b1:   0x02,
			b2:   0x62,
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := decodeCASLSystemCode(tt.b1, tt.b2)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if !tt.ok {
				return
			}
			if got != tt.want {
				t.Fatalf("got = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDecodeCASLRcomSurgardCode_TableDriven(t *testing.T) {
	tests := []struct {
		name string
		b1   byte
		b2   byte
		want caslDecodedEventCode
		ok   bool
	}{
		{
			name: "firmware static block",
			b1:   0x3B,
			b2:   0x05,
			want: caslDecodedEventCode{MessageKey: "PPK_SIM_4L"},
			ok:   true,
		},
		{
			name: "keyboard static code",
			b1:   0x08,
			b2:   0x2B,
			want: caslDecodedEventCode{MessageKey: "PROGRAMMING_CP_INTERNET"},
			ok:   true,
		},
		{
			name: "keyboard fallback offset",
			b1:   0x08,
			b2:   0x27,
			want: caslDecodedEventCode{MessageKey: "ID_HOZ", Number: 24, HasNumber: true},
			ok:   true,
		},
		{
			name: "static event table",
			b1:   0x00,
			b2:   0x69,
			want: caslDecodedEventCode{MessageKey: "OK_220"},
			ok:   true,
		},
		{
			name: "offset rule table",
			b1:   0x30,
			b2:   0x10,
			want: caslDecodedEventCode{MessageKey: "AD_DOOR_OP", Number: 1, HasNumber: true},
			ok:   true,
		},
		{
			name: "offset rule with positive offset",
			b1:   0x0C,
			b2:   0x05,
			want: caslDecodedEventCode{MessageKey: "ID_HOZ", Number: 54, HasNumber: true},
			ok:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := decodeCASLRcomSurgardCode(tt.b1, tt.b2)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if !tt.ok {
				return
			}
			if got != tt.want {
				t.Fatalf("got = %+v, want %+v", got, tt.want)
			}
		})
	}
}
