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
		want       float64
	}{
		{"two players default rates", 800, 20, 3, 2, 332},
		{"one player default rates", 800, 20, 3, 1, 664},
		{"two players no fees", 100, 0, 0, 2, 50},
		{"one player no fees", 100, 0, 0, 1, 100},
		{"rounds to two decimals", 333, 20, 3, 2, 138.2},
		{"zero players guarded", 800, 20, 3, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o := &repo.Order{
				Price:          c.price,
				CommissionRate: c.commission,
				BonusRate:      c.bonus,
				PlayerCount:    c.players,
			}
			got := o.Earnings()
			if math.Abs(got-c.want) > 0.001 {
				t.Fatalf("Earnings() = %v, want %v", got, c.want)
			}
		})
	}
}
