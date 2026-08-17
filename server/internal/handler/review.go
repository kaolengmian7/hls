package handler

// reviewEffects derives the score total, chicken-leg (bonus) qualification,
// and the resulting order complaint status from a review's two scores.
// Rule: total = attitude + technical; total>=8 qualifies for bonus;
// total<6 marks the order complained; total>=6 only reverts an
// auto-set "complained" back to "none" and never touches "resolved".
func reviewEffects(attitude, technical int, currentComplaint string) (total int, bonusQualified bool, newComplaint string) {
	total = attitude + technical
	bonusQualified = total >= 8
	switch {
	case total < 6:
		newComplaint = "complained"
	case currentComplaint == "complained":
		newComplaint = "none"
	default:
		newComplaint = currentComplaint
	}
	return
}
