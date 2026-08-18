package repo

import "math"

// round2 rounds a monetary value to two decimals.
func round2(v float64) float64 { return math.Round(v*100) / 100 }

// ComputeDeduction splits a gross salary into deposit repayment then fine
// repayment, flooring the net at zero. owed = outstanding deposit,
// fine = pending fine. Deposit is repaid before fine.
func ComputeDeduction(gross, owed, fine float64) (depositDeduct, fineDeduct, net float64) {
	if gross < 0 {
		gross = 0
	}
	if owed < 0 {
		owed = 0
	}
	if fine < 0 {
		fine = 0
	}
	depositDeduct = math.Min(gross, owed)
	remaining := gross - depositDeduct
	fineDeduct = math.Min(remaining, fine)
	net = remaining - fineDeduct
	return round2(depositDeduct), round2(fineDeduct), round2(net)
}
