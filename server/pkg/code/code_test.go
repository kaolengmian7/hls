package code

import "testing"

func TestIsValid(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"6位数字", "123456", true},
		{"首位为0仍合法", "012345", true},
		{"长度不足", "12345", false},
		{"长度过长", "1234567", false},
		{"含字母", "12345a", false},
		{"空", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValid(tt.in); got != tt.want {
				t.Errorf("IsValid(%q)=%v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestGeneratorAllocateUnique(t *testing.T) {
	g := NewGenerator()
	seen := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		c, err := g.Allocate()
		if err != nil {
			t.Fatalf("allocate failed: %v", err)
		}
		if !IsValid(c) {
			t.Fatalf("code %q invalid format", c)
		}
		if c[0] == '0' {
			t.Fatalf("code %q starts with 0", c)
		}
		if _, dup := seen[c]; dup {
			t.Fatalf("duplicate code %q", c)
		}
		seen[c] = struct{}{}
		if !g.Has(c) {
			t.Fatalf("Has(%q) should be true after Allocate", c)
		}
	}
}

func TestGeneratorRelease(t *testing.T) {
	g := NewGenerator()
	c, err := g.Allocate()
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}
	g.Release(c)
	if g.Has(c) {
		t.Fatalf("Has(%q) should be false after Release", c)
	}
}
