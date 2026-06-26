package ids

import "testing"

func TestPhoenixPPKNum(t *testing.T) {
	cases := []struct {
		panelID string
		want    int
		ok      bool
	}{
		{"L00022", 100022, true},
		{"l00022", 100022, true},
		{"L01000", 101000, true},
		{"L99999", 199999, true},
		{"L00000", 100000, true},
		{"L100000", 0, false}, // overflow
		{"00022", 100022, true},
		{"", 0, false},
		{"LABC", 0, false},
	}
	for _, c := range cases {
		got, ok := PhoenixPPKNum(c.panelID)
		if ok != c.ok || got != c.want {
			t.Errorf("PhoenixPPKNum(%q) = (%d, %v), want (%d, %v)", c.panelID, got, ok, c.want, c.ok)
		}
	}
}

func TestPhoenixPanelID(t *testing.T) {
	cases := []struct {
		ppkNum int
		want   string
		ok     bool
	}{
		{100022, "L00022", true},
		{101000, "L01000", true},
		{199999, "L99999", true},
		{100000, "L00000", true},
		{99999, "", false},
		{200000, "", false},
	}
	for _, c := range cases {
		got, ok := PhoenixPanelID(c.ppkNum)
		if ok != c.ok || got != c.want {
			t.Errorf("PhoenixPanelID(%d) = (%q, %v), want (%q, %v)", c.ppkNum, got, ok, c.want, c.ok)
		}
	}
}

func TestBridgePPKNum(t *testing.T) {
	cases := []struct {
		objectNumber string
		want         int
		ok           bool
	}{
		{"1", 1, true},
		{"10001", 10001, true},
		{"99999", 99999, true},
		{"100000", 0, false},
		{"0", 0, false},
		{"-1", 0, false},
		{"abc", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		got, ok := BridgePPKNum(c.objectNumber)
		if ok != c.ok || got != c.want {
			t.Errorf("BridgePPKNum(%q) = (%d, %v), want (%d, %v)", c.objectNumber, got, ok, c.want, c.ok)
		}
	}
}

func TestBridgeObjectNumber(t *testing.T) {
	cases := []struct {
		ppkNum int
		want   string
		ok     bool
	}{
		{10001, "10001", true},
		{1, "1", true},
		{99999, "99999", true},
		{0, "", false},
		{100000, "", false},
	}
	for _, c := range cases {
		got, ok := BridgeObjectNumber(c.ppkNum)
		if ok != c.ok || got != c.want {
			t.Errorf("BridgeObjectNumber(%d) = (%q, %v), want (%q, %v)", c.ppkNum, got, ok, c.want, c.ok)
		}
	}
}

func TestIsRanges(t *testing.T) {
	bridgeCases := []struct {
		num  int
		want bool
	}{
		{1, true}, {10001, true}, {99999, true},
		{0, false}, {100000, false},
	}
	for _, c := range bridgeCases {
		if got := IsBridgePPKNum(c.num); got != c.want {
			t.Errorf("IsBridgePPKNum(%d) = %v, want %v", c.num, got, c.want)
		}
	}

	phoenixCases := []struct {
		num  int
		want bool
	}{
		{100000, true}, {100022, true}, {199999, true},
		{99999, false}, {200000, false},
	}
	for _, c := range phoenixCases {
		if got := IsPhoenixPPKNum(c.num); got != c.want {
			t.Errorf("IsPhoenixPPKNum(%d) = %v, want %v", c.num, got, c.want)
		}
	}
}
