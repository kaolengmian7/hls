package repo_test

import (
	"testing"

	"hls.dev/server/internal/repo"
)

func TestComputeEarnings(t *testing.T) {
	// 800 @ 20% commission, bonus 3% not qualified, 1 player = 640
	if got := repo.ComputeEarnings(800, 20, 3, false, 1); got != 640 {
		t.Fatalf("want 640, got %v", got)
	}
	// bonus qualified adds 800*3% = 24 -> 664
	if got := repo.ComputeEarnings(800, 20, 3, true, 1); got != 664 {
		t.Fatalf("want 664, got %v", got)
	}
	// 2 players split
	if got := repo.ComputeEarnings(800, 20, 3, false, 2); got != 320 {
		t.Fatalf("want 320, got %v", got)
	}
	if got := repo.ComputeEarnings(800, 20, 3, false, 0); got != 0 {
		t.Fatalf("want 0 for zero players, got %v", got)
	}
}
