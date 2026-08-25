package repo_test

import (
	"math"
	"testing"

	"hls.dev/server/internal/repo"
)

func TestComputeDeduction(t *testing.T) {
	cases := []struct {
		name                         string
		gross, owed, fine            float64
		wantDeposit, wantFine, wantNet float64
	}{
		{"covers deposit and fine fully", 300, 100, 50, 100, 50, 150},
		{"only partial deposit", 80, 300, 40, 80, 0, 0},
		{"deposit filled, only fine", 200, 0, 50, 0, 50, 150},
		{"no owed no fine full pay", 120, 0, 0, 0, 0, 120},
		{"net floors at zero exact", 90, 60, 30, 60, 30, 0},
		{"fine larger than remaining", 100, 40, 999, 40, 60, 0},
		{"negative gross guarded", -5, 100, 100, 0, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, f, n := repo.ComputeDeduction(c.gross, c.owed, c.fine)
			if math.Abs(d-c.wantDeposit) > 0.001 || math.Abs(f-c.wantFine) > 0.001 || math.Abs(n-c.wantNet) > 0.001 {
				t.Fatalf("ComputeDeduction(%v,%v,%v) = (%v,%v,%v), want (%v,%v,%v)",
					c.gross, c.owed, c.fine, d, f, n, c.wantDeposit, c.wantFine, c.wantNet)
			}
		})
	}
}
