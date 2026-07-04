package handler

import "testing"

func TestIsTestPhone(t *testing.T) {
	cases := []struct {
		phone string
		want  bool
	}{
		{"900006000", true},
		{"900006099", true},
		{"900006100", true},
		{"900006199", true},
		{"900006200", true},
		{"900006299", true},
		{"900006300", false},
		{"13800138000", false},
		{"9000060", false},
		{"9000060001", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isTestPhone(c.phone); got != c.want {
			t.Errorf("isTestPhone(%q) = %v, want %v", c.phone, got, c.want)
		}
	}
}
