package handler

import "testing"

func TestReviewEffects(t *testing.T) {
	cases := []struct {
		name          string
		attitude      int
		technical     int
		current       string
		wantTotal     int
		wantQualified bool
		wantComplaint string
	}{
		{"perfect keeps none", 5, 5, "none", 10, true, "none"},
		{"exactly 8 qualifies", 4, 4, "none", 8, true, "none"},
		{"total 7 neither", 4, 3, "none", 7, false, "none"},
		{"total 6 boundary no complaint", 3, 3, "none", 6, false, "none"},
		{"below 6 sets complained", 2, 3, "none", 5, false, "complained"},
		{"recovery reverts complained", 4, 3, "complained", 7, false, "none"},
		{"recovery does not touch resolved", 4, 4, "resolved", 8, true, "resolved"},
		{"low score overrides resolved", 2, 2, "resolved", 4, false, "complained"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			total, qualified, complaint := reviewEffects(c.attitude, c.technical, c.current)
			if total != c.wantTotal || qualified != c.wantQualified || complaint != c.wantComplaint {
				t.Fatalf("reviewEffects(%d,%d,%q) = (%d,%v,%q), want (%d,%v,%q)",
					c.attitude, c.technical, c.current, total, qualified, complaint,
					c.wantTotal, c.wantQualified, c.wantComplaint)
			}
		})
	}
}
