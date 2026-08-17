package repo_test

import (
	"math"
	"testing"

	"hls.dev/server/internal/repo"
)

func TestOrderEarnings(t *testing.T) {
	cases := []struct {
		name       string
		price      float64
		commission float64
		bonus      float64
		players    int
		qualified  bool
		want       float64
	}{
		{"qualified two players default rates", 800, 20, 3, 2, true, 332},
		{"qualified one player default rates", 800, 20, 3, 1, true, 664},
		{"unqualified two players drops bonus", 800, 20, 3, 2, false, 320},
		{"unqualified one player drops bonus", 800, 20, 3, 1, false, 640},
		{"two players no fees", 100, 0, 0, 2, true, 50},
		{"one player no fees unqualified", 100, 0, 0, 1, false, 100},
		{"qualified rounds to two decimals", 333, 20, 3, 2, true, 138.2},
		{"zero players guarded", 800, 20, 3, 0, true, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o := &repo.Order{
				Price:          c.price,
				CommissionRate: c.commission,
				BonusRate:      c.bonus,
				PlayerCount:    c.players,
				BonusQualified: c.qualified,
			}
			got := o.Earnings()
			if math.Abs(got-c.want) > 0.001 {
				t.Fatalf("Earnings() = %v, want %v", got, c.want)
			}
		})
	}
}
