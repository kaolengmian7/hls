package game

import (
	"os"
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
	origURLs := make([]string, len(lotteryPool))
	for i := range lotteryPool {
		origURLs[i] = lotteryPool[i].ImageURL
		lotteryPool[i].ImageURL = "https://cdn.example.com/prizes/" + lotteryPool[i].ID + ".webp"
	}
	t.Cleanup(func() {
		for i := range lotteryPool {
			lotteryPool[i].ImageURL = origURLs[i]
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

func TestMain(m *testing.M) {
	if err := InitPool(defaultTestPool()); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestInitPool(t *testing.T) {
	t.Run("nil pool returns error", func(t *testing.T) {
		if err := InitPool(nil); err == nil {
			t.Fatal("expected error for nil pool")
		}
	})
	t.Run("empty pool returns error", func(t *testing.T) {
		if err := InitPool([]LotteryItem{}); err == nil {
			t.Fatal("expected error for empty pool")
		}
	})
	t.Run("valid pool sets lotteryPool", func(t *testing.T) {
		orig := lotteryPool
		t.Cleanup(func() { lotteryPool = orig })
		items := []LotteryItem{
			{ID: "X", Name: "X Item", Rarity: rarityHigh, Value: 100, Weight: 1.0,
				ImageURL: "https://cdn.example.com/x.webp"},
		}
		if err := InitPool(items); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(lotteryPool) != 1 {
			t.Fatalf("want 1 item, got %d", len(lotteryPool))
		}
		if lotteryPool[0].ID != "X" {
			t.Errorf("ID=%s, want X", lotteryPool[0].ID)
		}
		// 确保 InitPool 做了拷贝，外部修改不影响包级 pool
		items[0].ID = "mutated"
		if lotteryPool[0].ID != "X" {
			t.Error("InitPool must copy items, not reference input slice")
		}
	})
}

// defaultTestPool 返回与生产池一致的 51 件奖品，供 TestMain 初始化。
func defaultTestPool() []LotteryItem {
	return []LotteryItem{
		{ID: "TEAR_OF_THE_OCEAN", Name: "海洋之泪", Rarity: rarityHigh, Value: 2800, Weight: 0.05,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/tear-of-the-ocean.webp"},
		{ID: "HEART_OF_AFRICA", Name: "非洲之心", Rarity: rarityHigh, Value: 1300, Weight: 0.1,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/heart-of-africa.webp"},
		{ID: "PORTABLE_VENTILATOR", Name: "复苏呼吸机", Rarity: rarityHigh, Value: 530, Weight: 0.3,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/portable-ventilator.webp"},
		{ID: "MICRO_REACTOR", Name: "微型反应炉", Rarity: rarityHigh, Value: 470, Weight: 0.4,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/micro-reactor.webp"},
		{ID: "ARMORED_VEHICLE_BATTERY", Name: "装甲车电池", Rarity: rarityHigh, Value: 380, Weight: 0.55,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/armored-vehicle-battery.webp"},
		{ID: "ECMO", Name: "ECMO", Rarity: rarityHigh, Value: 300, Weight: 0.7,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/ecmo.webp"},
		{ID: "RADAR_PORTABLE", Name: "便携军用雷达", Rarity: rarityHigh, Value: 215, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/radar-portable.webp"},
		{ID: "MANDEL_SUPERCOMPUTER", Name: "曼德尔超算单元", Rarity: rarityHigh, Value: 200, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/mandel-supercomputer.webp"},
		{ID: "DISK_ARRAY_HS", Name: "高速磁盘阵列", Rarity: rarityHigh, Value: 170, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/disk-array-hs.webp"},
		{ID: "GOLDEN_CROWN_OF_TEARS", Name: "万金泪冠", Rarity: rarityHigh, Value: 230, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/golden-crown-of-tears.webp"},
		{ID: "VERTICAL_HORIZONTAL", Name: "纵横", Rarity: rarityHigh, Value: 240, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/vertical-horizontal.webp"},
		{ID: "IMPRESSIONIST_PAINTING", Name: "印象派名画", Rarity: rarityHigh, Value: 210, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/impressionist-painting.webp"},
		{ID: "ROBOT_VAC_MOP", Name: "扫拖一体机器人", Rarity: rarityHigh, Value: 200, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/robot-vac-mop.webp"},
		{ID: "REINFORCED_CARBON_FIBER_PLATE", Name: "强化碳纤维板", Rarity: rarityHigh, Value: 200, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/reinforced-carbon-fiber-plate.webp"},
		{ID: "FLIGHT_RECORDER", Name: "飞行记录仪", Rarity: rarityHigh, Value: 170, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/flight-recorder.webp"},
		{ID: "FAMOUS_KILN_PORCELAIN", Name: "名窑瓷器", Rarity: rarityHigh, Value: 150, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/famous-kiln-porcelain.webp"},
		{ID: "BATTERY_PACK", Name: "动力电池组", Rarity: rarityHigh, Value: 150, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/battery-pack.webp"},
		{ID: "MIL_SHELL", Name: "军用炮弹", Rarity: rarityHigh, Value: 150, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/mil-shell.webp"},
		{ID: "DISK_ARRAY_HS_COMPACT", Name: "高速磁盘阵列", Rarity: rarityHigh, Value: 150, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/disk-array-hs-compact.webp"},
		{ID: "AED", Name: "自动体外除颤仪", Rarity: rarityHigh, Value: 140, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/aed.webp"},
		{ID: "POWERFUL_VACUUM", Name: "强力吸尘器", Rarity: rarityHigh, Value: 140, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/powerful-vacuum.webp"},
		{ID: "BLADE_SERVER", Name: "刀片服务器", Rarity: rarityHigh, Value: 140, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/blade-server.webp"},
		{ID: "IFV_MODEL_DELUXE", Name: "步战车模型", Rarity: rarityHigh, Value: 220, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/ifv-model-deluxe.webp"},
		{ID: "IFV_MODEL", Name: "步战车模型", Rarity: rarityMedium, Value: 130, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/ifv-model.webp"},
		{ID: "MIL_DRONE", Name: "军用无人机", Rarity: rarityMedium, Value: 50, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/mil-drone.webp"},
		{ID: "MIL_INFO_TERMINAL", Name: "军用信息终端", Rarity: rarityMedium, Value: 38, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/mil-info-terminal.webp"},
		{ID: "CHAMPAGNE", Name: "香槟", Rarity: rarityMedium, Value: 34, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/champagne.webp"},
		{ID: "GPU", Name: "显卡", Rarity: rarityMedium, Value: 33, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/gpu.webp"},
		{ID: "QUANTUM_STORAGE", Name: "量子存储", Rarity: rarityMedium, Value: 30, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/quantum-storage.webp"},
		{ID: "GOLD_BAR", Name: "万足金条", Rarity: rarityMedium, Value: 30, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/gold-bar.webp"},
		{ID: "FOSSIL", Name: "化石", Rarity: rarityMedium, Value: 35, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/fossil.webp"},
		{ID: "GOLDEN_GAZELLE", Name: "黄金瞪羚", Rarity: rarityMedium, Value: 40, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/golden-gazelle.webp"},
		{ID: "ARCHIVED_AUDIO_SOURCE", Name: "已封存音源", Rarity: rarityMedium, Value: 50, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/archived-audio-source.webp"},
		{ID: "GOLDEN_CROCODILE_BUST", Name: "黄金鳄鱼头雕像", Rarity: rarityMedium, Value: 50, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/golden-crocodile-bust.webp"},
		{ID: "TIANYUAN_DIFANG", Name: "天圆地方", Rarity: rarityMedium, Value: 60, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/tianyuan-difang.webp"},
		{ID: "CAMERA", Name: "摄影机", Rarity: rarityMedium, Value: 65, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/camera.webp"},
		{ID: "MUSKET", Name: "滑膛枪", Rarity: rarityMedium, Value: 70, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/musket.webp"},
		{ID: "MIL_RADIO", Name: "军用电台", Rarity: rarityMedium, Value: 100, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/mil-radio.webp"},
		{ID: "ELECTRONIC_ANKLET", Name: "电子脚镣", Rarity: rarityMedium, Value: 100, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/electronic-anklet.webp"},
		{ID: "MEDICAL_ROBOT", Name: "医疗机器人", Rarity: rarityMedium, Value: 120, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/medical-robot.webp"},
		{ID: "LAPTOP", Name: "笔记本电脑", Rarity: rarityMedium, Value: 120, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/laptop.webp"},
		{ID: "CLOUD_STORAGE_ARRAY", Name: "云储存阵列", Rarity: rarityMedium, Value: 130, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/cloud-storage-array.webp"},
		{ID: "VENTILATOR", Name: "呼吸机", Rarity: rarityMedium, Value: 45, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/ventilator.webp"},
		{ID: "GPS_RECEIVER", Name: "定位接收器", Rarity: rarityLow, Value: 18, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/gps-receiver.webp"},
		{ID: "STAR_SENSOR", Name: "恒星敏感器", Rarity: rarityLow, Value: 18, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/star-sensor.webp"},
		{ID: "CAVIAR", Name: "鱼子酱", Rarity: rarityLow, Value: 17, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/caviar.webp"},
		{ID: "GILT_CARD", Name: "鎏金卡牌", Rarity: rarityLow, Value: 20, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/gilt-card.webp"},
		{ID: "ROLEX", Name: "劳力士", Rarity: rarityLow, Value: 20, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/rolex.webp"},
		{ID: "SAYYID_POCKET_WATCH", Name: "赛义德的怀表", Rarity: rarityLow, Value: 20, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/sayyid-pocket-watch.webp"},
		{ID: "ARCHIVED_EXPERIMENT_DATA", Name: "实验数据", Rarity: rarityLow, Value: 20, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/archived-experiment-data.webp"},
		{ID: "ULTRASONIC_CUTTER", Name: "超声波切割刀", Rarity: rarityLow, Value: 20, Weight: 6.08,
			ImageURL: "https://hls-1307859563.cos.ap-shanghai.myqcloud.com/red/ultrasonic-cutter.webp"},
	}
}
