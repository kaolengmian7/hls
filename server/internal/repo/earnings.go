package repo

import "math"

// ComputeEarnings 计算单人应发工资，四舍五入两位。
// 公式：(price*(1-commission%/100) + (qualified ? price*bonus%/100 : 0)) / playerCount。
func ComputeEarnings(price, commissionRate, bonusRate float64, bonusQualified bool, playerCount int) float64 {
	if playerCount <= 0 {
		return 0
	}
	net := price * (1 - commissionRate/100)
	bonus := 0.0
	if bonusQualified {
		bonus = price * bonusRate / 100
	}
	per := (net + bonus) / float64(playerCount)
	return math.Round(per*100) / 100
}
