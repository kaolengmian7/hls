package game

import (
	"testing"

	"hls.dev/server/pkg/errors"
)

func TestCostForTimes(t *testing.T) {
	tests := []struct {
		name    string
		times   int
		wantCst int
		wantErr bool
	}{
		{"单抽", 1, SingleDrawCost, false},
		{"非法十连", 10, 0, true},
		{"非法 5 连", 5, 0, true},
		{"零次", 0, 0, true},
		{"负数", -1, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CostForTimes(tt.times)
			if (err != nil) != tt.wantErr {
				t.Errorf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.wantCst {
				t.Errorf("cost=%d, want %d", got, tt.wantCst)
			}
		})
	}
}

func TestDrawLotterySingle(t *testing.T) {
	const current = 2500
	records, newGuarantee, err := DrawLottery("p1", 1, current, 1.0)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expect 1 record, got %d", len(records))
	}
	r := records[0]
	if r.PlayerID != "p1" {
		t.Errorf("playerID=%s", r.PlayerID)
	}
	if r.Cost != SingleDrawCost {
		t.Errorf("cost=%d", r.Cost)
	}
	if r.Item.Name == "" || r.Item.Rarity == "" {
		t.Errorf("item missing name/rarity")
	}
	if r.Item.Value <= 0 {
		t.Errorf("item value must be positive, got %d", r.Item.Value)
	}
	if r.GuaranteeDelta != r.Item.Value {
		t.Errorf("delta(%d) should equal item value(%d)", r.GuaranteeDelta, r.Item.Value)
	}
	if r.GuaranteeBefore != current {
		t.Errorf("before=%d want %d", r.GuaranteeBefore, current)
	}
	if r.GuaranteeAfter != current+r.GuaranteeDelta {
		t.Errorf("after=%d want %d", r.GuaranteeAfter, current+r.GuaranteeDelta)
	}
	if newGuarantee != r.GuaranteeAfter {
		t.Errorf("returned guarantee(%d) != record.after(%d)", newGuarantee, r.GuaranteeAfter)
	}
}

func TestDrawLotteryInvalidTimes(t *testing.T) {
	tests := []struct {
		name  string
		times int
	}{
		{"十连已下线", 10},
		{"任意多连", 3},
		{"零次", 0},
		{"负数", -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, guarantee, err := DrawLottery("p1", tt.times, 2500, 1.0)
			if err == nil {
				t.Fatalf("expect error for times=%d", tt.times)
			}
			if records != nil {
				t.Errorf("records should be nil on error, got %d items", len(records))
			}
			if guarantee != 2500 {
				t.Errorf("guarantee should remain unchanged on error, got %d", guarantee)
			}
		})
	}
}

func TestDrawLotteryRollbackOnOverflow(t *testing.T) {
	// 任意奖品 Value 都为正整数，currentGuarantee 取上限 → 累加必越界 → 回滚。
	records, guarantee, err := DrawLottery("p1", 1, MaxGuaranteedAmount, 1.0)
	if err == nil {
		t.Fatalf("expect overflow error at currentGuarantee=MaxGuaranteedAmount")
	}
	if !errors.IsExposable(err) {
		t.Fatalf("error should be exposable")
	}
	if err.Error() != "INVALID_GUARANTEE" {
		t.Fatalf("reason=%q want INVALID_GUARANTEE", err.Error())
	}
	if records != nil {
		t.Errorf("records should be nil on rollback, got %d items", len(records))
	}
	if guarantee != MaxGuaranteedAmount {
		t.Errorf("guarantee must remain at %d on rollback, got %d", MaxGuaranteedAmount, guarantee)
	}
}

func TestLotteryPoolDefinition(t *testing.T) {
	// 奖品池共 51 件：prizeId 唯一、Value > 0、Weight > 0。
	if len(lotteryPool) != 51 {
		t.Fatalf("expect 51 prizes in pool, got %d", len(lotteryPool))
	}
	seen := make(map[string]bool, len(lotteryPool))
	for _, it := range lotteryPool {
		if it.ID == "" || it.Name == "" || it.Rarity == "" {
			t.Errorf("prize %+v missing required fields", it)
		}
		if it.Value <= 0 {
			t.Errorf("prize %s value must be > 0, got %d", it.ID, it.Value)
		}
		if it.Weight <= 0 {
			t.Errorf("prize %s weight must be > 0, got %f", it.ID, it.Weight)
		}
		if seen[it.ID] {
			t.Errorf("duplicate prizeId: %s", it.ID)
		}
		seen[it.ID] = true
	}
	// 抽样确认文档约定的关键奖品存在且价值正确。
	want := map[string]int{
		"RADAR_PORTABLE":       215,
		"MANDEL_SUPERCOMPUTER": 200,
		"DISK_ARRAY_HS":        170,
		"MIL_DRONE":            50,
		"MIL_INFO_TERMINAL":    38,
		"CHAMPAGNE":            34,
		"GPU":                  33,
		"QUANTUM_STORAGE":      30,
		"GPS_RECEIVER":         18,
		"STAR_SENSOR":          18,
		"CAVIAR":               17,
	}
	got := make(map[string]int, len(lotteryPool))
	for _, it := range lotteryPool {
		got[it.ID] = it.Value
	}
	for id, v := range want {
		if got[id] != v {
			t.Errorf("prize %s value=%d, want %d", id, got[id], v)
		}
	}
}

func TestDrawLotteryPropagatesImageURL(t *testing.T) {
	// 给每件奖品设可预测的 URL，保证无论 roll 到哪件都能断言。
	for i := range lotteryPool {
		lotteryPool[i].ImageURL = "https://cdn.example.com/prizes/" + lotteryPool[i].ID + ".webp"
	}
	t.Cleanup(func() {
		for i := range lotteryPool {
			lotteryPool[i].ImageURL = ""
		}
	})

	records, _, err := DrawLottery("player-test", 1, 0, 1.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantURL := "https://cdn.example.com/prizes/" + records[0].Item.ID + ".webp"
	if records[0].Item.ImageURL != wantURL {
		t.Errorf("ImageURL mismatch: got %q, want %q", records[0].Item.ImageURL, wantURL)
	}
}

func TestRollOneDistribution(t *testing.T) {
	pool := []LotteryItem{
		{ID: "a", Name: "a", Rarity: "x", Value: 1, Weight: 1},
		{ID: "b", Name: "b", Rarity: "x", Value: 1, Weight: 99},
	}
	counts := map[string]int{}
	for i := 0; i < 5000; i++ {
		item, err := rollOne(pool)
		if err != nil {
			t.Fatal(err)
		}
		counts[item.ID]++
	}
	if counts["b"] <= counts["a"] {
		t.Errorf("biased pool wrong: a=%d b=%d", counts["a"], counts["b"])
	}
}

func TestGetPool(t *testing.T) {
	pool := GetPool()
	if len(pool) != len(lotteryPool) {
		t.Fatalf("GetPool len=%d, want %d", len(pool), len(lotteryPool))
	}
	for i, item := range pool {
		src := lotteryPool[i]
		if item.ID != src.ID {
			t.Errorf("[%d] ID=%s, want %s", i, item.ID, src.ID)
		}
		if item.Name != src.Name {
			t.Errorf("[%d] Name=%s, want %s", i, item.Name, src.Name)
		}
		if item.Rarity != src.Rarity {
			t.Errorf("[%d] Rarity=%s, want %s", i, item.Rarity, src.Rarity)
		}
		if item.Value != src.Value {
			t.Errorf("[%d] Value=%d, want %d", i, item.Value, src.Value)
		}
		if item.ImageURL != src.ImageURL {
			t.Errorf("[%d] ImageURL=%s, want %s", i, item.ImageURL, src.ImageURL)
		}
	}
}

func TestApplyHighBoost(t *testing.T) {
	pool := []LotteryItem{
		{ID: "h1", Name: "H1", Rarity: rarityHigh, Value: 100, Weight: 2.0},
		{ID: "m1", Name: "M1", Rarity: rarityMedium, Value: 50, Weight: 6.0},
		{ID: "l1", Name: "L1", Rarity: rarityLow, Value: 10, Weight: 6.0},
	}
	boosted := applyHighBoost(pool, 5.0)

	// HIGH weight multiplied
	if boosted[0].Weight != 10.0 {
		t.Errorf("HIGH weight got %f, want 10.0", boosted[0].Weight)
	}
	// MEDIUM unchanged
	if boosted[1].Weight != 6.0 {
		t.Errorf("MEDIUM weight got %f, want 6.0", boosted[1].Weight)
	}
	// LOW unchanged
	if boosted[2].Weight != 6.0 {
		t.Errorf("LOW weight got %f, want 6.0", boosted[2].Weight)
	}
	// original pool not mutated
	if pool[0].Weight != 2.0 {
		t.Errorf("original HIGH weight mutated, got %f", pool[0].Weight)
	}
}

func TestDrawLottery_VIPBoost(t *testing.T) {
	// With a huge boost, HIGH items must dominate.
	const runs = 1000
	highCount := 0
	for i := 0; i < runs; i++ {
		records, _, err := DrawLottery("p1", 1, 0, 1000.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if records[0].Item.Rarity == rarityHigh {
			highCount++
		}
	}
	ratio := float64(highCount) / runs
	if ratio < 0.95 {
		t.Errorf("HIGH ratio=%.2f with boost=1000, want >= 0.95", ratio)
	}
}
