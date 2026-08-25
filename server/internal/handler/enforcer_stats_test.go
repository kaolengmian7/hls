package handler

import (
	"database/sql"
	"testing"

	"hls.dev/server/internal/repo"
)

func TestComputeEnforcerStats(t *testing.T) {
	users := []*repo.UserSummary{
		{ID: 1, Username: "hlsaaa", Nickname: "打手甲", Role: "enforcer"},
		{ID: 2, Username: "svc", Nickname: "客服", Role: "service"},
	}
	orders := []*repo.Order{
		{Code: "o1", EnforcerAID: sql.NullInt64{Int64: 1, Valid: true}},
		{Code: "o2", EnforcerAID: sql.NullInt64{Int64: 1, Valid: true}},
		{Code: "o3", EnforcerAID: sql.NullInt64{Int64: 1, Valid: true}},
	}
	reviews := map[string]*repo.ReviewSummary{
		"o1": {OrderCode: "o1", Attitude: 5, Technical: 5}, // 10 -> good
		"o2": {OrderCode: "o2", Attitude: 2, Technical: 3}, // 5  -> bad
		// o3 unreviewed
	}
	accounts := map[int64]*repo.EnforcerAccount{
		1: {UserID: 1, DepositPaid: 120, PendingFine: 40},
	}
	got := computeEnforcerStats(users, orders, reviews, accounts, 300)
	if len(got) != 1 {
		t.Fatalf("expected 1 enforcer, got %d", len(got))
	}
	s := got[0]
	if s.OrderCount != 3 || s.GoodCount != 1 || s.BadCount != 1 || s.ReviewedCount != 2 {
		t.Fatalf("counts wrong: %+v", s)
	}
	if s.GoodRate == nil || *s.GoodRate != 0.5 {
		t.Fatalf("goodRate wrong: %+v", s.GoodRate)
	}
	if s.DepositRequired != 300 || s.DepositPaid != 120 || s.DepositOwed != 180 || s.PendingFine != 40 {
		t.Fatalf("account fields wrong: %+v", s)
	}
}

func TestComputeEnforcerStats_NoReviews(t *testing.T) {
	users := []*repo.UserSummary{{ID: 1, Nickname: "打手", Role: "enforcer"}}
	orders := []*repo.Order{{Code: "o1", EnforcerAID: sql.NullInt64{Int64: 1, Valid: true}}}
	got := computeEnforcerStats(users, orders, map[string]*repo.ReviewSummary{}, map[int64]*repo.EnforcerAccount{}, 300)
	if got[0].GoodRate != nil {
		t.Fatalf("expected nil goodRate, got %v", *got[0].GoodRate)
	}
	if got[0].DepositRequired != 300 || got[0].DepositOwed != 300 {
		t.Fatalf("default deposit wrong: %+v", got[0])
	}
}
