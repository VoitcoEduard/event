package main

import "testing"

func TestListenPort(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  int
		valid bool
	}{
		{"", 8080, true}, {"8081", 8081, true}, {"1", 1, true}, {"65535", 65535, true},
		{"0", 0, false}, {"65536", 0, false}, {"abc", 0, false}, {"-1", 0, false},
	} {
		got, err := listenPort(tc.input)
		if got != tc.want || (err == nil) != tc.valid {
			t.Errorf("listenPort(%q) = %d, %v", tc.input, got, err)
		}
	}
}
