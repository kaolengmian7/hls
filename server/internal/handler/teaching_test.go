package handler

import "testing"

func TestParseTeachingPath(t *testing.T) {
	code, rest := parseTeachingPath("/api/teaching-orders/JX123/scorecards")
	if code != "JX123" || len(rest) != 1 || rest[0] != "scorecards" {
		t.Fatalf("unexpected: %q %v", code, rest)
	}
	code, rest = parseTeachingPath("/api/teaching-orders/JX9")
	if code != "JX9" || len(rest) != 0 {
		t.Fatalf("unexpected: %q %v", code, rest)
	}
}

func TestValidateScores(t *testing.T) {
	valid := map[string]bool{"pre_aim": true, "peek": true}
	if validateScores(map[string]int{"pre_aim": 8}, valid) != "" {
		t.Fatal("valid scores should pass")
	}
	if validateScores(map[string]int{"pre_aim": 11}, valid) == "" {
		t.Fatal("score 11 should fail")
	}
	if validateScores(map[string]int{"nope": 5}, valid) == "" {
		t.Fatal("unknown key should fail")
	}
}

func TestHoursToMinutes(t *testing.T) {
	if hoursToMinutes(1.5) != 90 || hoursToMinutes(6) != 360 {
		t.Fatalf("bad conversion")
	}
}
