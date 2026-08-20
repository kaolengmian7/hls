package handler

import "testing"

func TestBonusQualifies(t *testing.T) {
	if !BonusQualifies(5, 3) { // total 8
		t.Fatal("8 should qualify")
	}
	if BonusQualifies(4, 3) { // total 7
		t.Fatal("7 should not qualify")
	}
}
