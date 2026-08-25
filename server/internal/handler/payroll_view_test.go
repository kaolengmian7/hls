package handler

import (
	"testing"

	"hls.dev/server/internal/repo"
)

func TestProjectPayrollView(t *testing.T) {
	item := &repo.PayrollItem{ID: 7, OrderCode: "o1", EnforcerName: "打手甲", Status: "pending"}
	order := &repo.Order{Code: "o1", Title: "上分单", Price: 800, CommissionRate: 20, BonusRate: 3, PlayerCount: 1}
	src := &payrollSource{Title: order.Title, Gross: order.Earnings()}
	acc := &repo.EnforcerAccount{UserID: 1, DepositPaid: 0, PendingFine: 50}
	v := projectPayrollView(item, src, acc, 300)
	// gross = 640 (unqualified), deposit 300, fine 50, net 290
	if v.Gross != 640 || v.DepositDeducted != 300 || v.FineDeducted != 50 || v.NetPaid != 290 {
		t.Fatalf("unexpected projection: %+v", v)
	}
	if v.OrderTitle != "上分单" || v.ID != 7 || v.Status != "pending" {
		t.Fatalf("unexpected meta: %+v", v)
	}
}
