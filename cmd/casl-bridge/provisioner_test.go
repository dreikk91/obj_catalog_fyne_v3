package main

import "testing"

func TestStringFromCASL(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{name: "string", in: " 42 ", want: "42"},
		{name: "float integer", in: float64(42), want: "42"},
		{name: "int", in: 42, want: "42"},
		{name: "int64", in: int64(42), want: "42"},
		{name: "nil", in: nil, want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := stringFromCASL(tc.in); got != tc.want {
				t.Fatalf("stringFromCASL(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestIntFromCASL(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want int
	}{
		{name: "string", in: " 300 ", want: 300},
		{name: "float", in: float64(300), want: 300},
		{name: "int", in: 300, want: 300},
		{name: "int64", in: int64(300), want: 300},
		{name: "bad string", in: "bad", want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := intFromCASL(tc.in); got != tc.want {
				t.Fatalf("intFromCASL(%v) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}
