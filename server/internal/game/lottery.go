package game

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"time"

	"hls.dev/server/internal/protocol"
	"hls.dev/server/pkg/errors"
)

// 抽奖消耗常量。
//
// 当前仅支持单抽（times == 1）；如需扩展多连抽，请在此追加新的常量与
// CostForTimes / DrawLottery 的分支，并同步前端 IntentMap.DRAW_LOTTERY。
const (
	SingleDrawCost = 2
)

// 暴露给上层的 reason 字符串（直接写入 ACTION_REJECTED.reason）。
// 与 settlement 保持同样的对外契约。
const (
	lotteryReasonInvalidGuarantee = "INVALID_GUARANTEE"
)

// LotteryItem 奖池物品定义。
//
// Value 为奖品估值（单位 W = 万），正整数。抽中后服务端会把 Value
// 累加到房间 GuaranteedAmount，参见 DrawLottery。
type LotteryItem struct {
	ID     string
	Name   string
	Rarity string
	Value  int
	Weight float64
}

// 稀有度档位常量。Weight 是后端权重，仅服务端可见。
//
// 设计：价值越高的奖品权重越低，但服务端权威配置可独立调整。
const (
	rarityHigh   = "HIGH"
	rarityMedium = "MEDIUM"
	rarityLow    = "LOW"
)

// lotteryPool 老板抽奖奖品池。
//
// 共 42 件奖品，价值合计 5403W。每件奖品的 Value 在抽中时累加到房间
// GuaranteedAmount；如累加后 newGuarantee > MaxGuaranteedAmount，
// 则整次抽奖回滚（金币不扣、保底不变、不下发 LOTTERY_DRAWN）。
var lotteryPool = []LotteryItem{
	{ID: "PORTABLE_VENTILATOR", Name: "复苏呼吸机", Rarity: rarityHigh, Value: 530, Weight: 0.3},
	{ID: "MICRO_REACTOR", Name: "微型反应炉", Rarity: rarityHigh, Value: 470, Weight: 0.4},
	{ID: "ARMORED_VEHICLE_BATTERY", Name: "装甲车电池", Rarity: rarityHigh, Value: 380, Weight: 0.55},
	{ID: "ECMO", Name: "ECMO", Rarity: rarityHigh, Value: 300, Weight: 0.7},
	{ID: "RADAR_PORTABLE", Name: "便携军用雷达", Rarity: rarityHigh, Value: 215, Weight: 1},
	{ID: "MANDEL_SUPERCOMPUTER", Name: "曼德尔超算单元", Rarity: rarityHigh, Value: 200, Weight: 1.2},
	{ID: "DISK_ARRAY_HS", Name: "高速磁盘阵列", Rarity: rarityHigh, Value: 170, Weight: 1.5},
	{ID: "GOLDEN_CROWN_OF_TEARS", Name: "万金泪冠", Rarity: rarityHigh, Value: 230, Weight: 0.9},
	{ID: "VERTICAL_HORIZONTAL", Name: "纵横", Rarity: rarityHigh, Value: 240, Weight: 0.8},
	{ID: "IMPRESSIONIST_PAINTING", Name: "印象派名画", Rarity: rarityHigh, Value: 210, Weight: 1.1},
	{ID: "ROBOT_VAC_MOP", Name: "扫拖一体机器人", Rarity: rarityHigh, Value: 200, Weight: 1.2},
	{ID: "REINFORCED_CARBON_FIBER_PLATE", Name: "强化碳纤维板", Rarity: rarityHigh, Value: 200, Weight: 1.2},
	{ID: "FAMOUS_KILN_PORCELAIN", Name: "名窑瓷器", Rarity: rarityHigh, Value: 150, Weight: 1.7},
	{ID: "BATTERY_PACK", Name: "动力电池组", Rarity: rarityHigh, Value: 150, Weight: 1.7},
	{ID: "MIL_SHELL", Name: "军用炮弹", Rarity: rarityHigh, Value: 150, Weight: 1.7},
	{ID: "AED", Name: "自动体外除颤仪", Rarity: rarityHigh, Value: 140, Weight: 1.9},
	{ID: "POWERFUL_VACUUM", Name: "强力吸尘器", Rarity: rarityHigh, Value: 140, Weight: 1.9},
	{ID: "IFV_MODEL_DELUXE", Name: "步战车模型", Rarity: rarityHigh, Value: 220, Weight: 1},
	{ID: "MIL_DRONE", Name: "军用无人机", Rarity: rarityMedium, Value: 50, Weight: 6},
	{ID: "MIL_INFO_TERMINAL", Name: "军用信息终端", Rarity: rarityMedium, Value: 38, Weight: 8},
	{ID: "CHAMPAGNE", Name: "香槟", Rarity: rarityMedium, Value: 34, Weight: 9},
	{ID: "GPU", Name: "显卡", Rarity: rarityMedium, Value: 33, Weight: 9},
	{ID: "QUANTUM_STORAGE", Name: "量子存储", Rarity: rarityMedium, Value: 30, Weight: 10},
	{ID: "GOLD_BAR", Name: "万足金条", Rarity: rarityMedium, Value: 30, Weight: 10},
	{ID: "FOSSIL", Name: "化石", Rarity: rarityMedium, Value: 35, Weight: 8.5},
	{ID: "GOLDEN_GAZELLE", Name: "黄金瞪羚", Rarity: rarityMedium, Value: 40, Weight: 7.5},
	{ID: "ARCHIVED_AUDIO_SOURCE", Name: "已封存音源", Rarity: rarityMedium, Value: 50, Weight: 6},
	{ID: "GOLDEN_CROCODILE_BUST", Name: "黄金鳄鱼头雕像", Rarity: rarityMedium, Value: 50, Weight: 6},
	{ID: "TIANYUAN_DIFANG", Name: "天圆地方", Rarity: rarityMedium, Value: 60, Weight: 5},
	{ID: "MUSKET", Name: "滑膛枪", Rarity: rarityMedium, Value: 70, Weight: 4.5},
	{ID: "MEDICAL_ROBOT", Name: "医疗机器人", Rarity: rarityMedium, Value: 120, Weight: 2.6},
	{ID: "IFV_MODEL", Name: "步战车模型", Rarity: rarityMedium, Value: 130, Weight: 2.3},
	{ID: "CLOUD_STORAGE_ARRAY", Name: "云储存阵列", Rarity: rarityMedium, Value: 130, Weight: 2.3},
	{ID: "VENTILATOR", Name: "呼吸机", Rarity: rarityMedium, Value: 45, Weight: 7},
	{ID: "GPS_RECEIVER", Name: "定位接收器", Rarity: rarityLow, Value: 18, Weight: 17},
	{ID: "STAR_SENSOR", Name: "恒星敏感器", Rarity: rarityLow, Value: 18, Weight: 17},
	{ID: "CAVIAR", Name: "鱼子酱", Rarity: rarityLow, Value: 17, Weight: 20},
	{ID: "GILT_CARD", Name: "鎏金卡牌", Rarity: rarityLow, Value: 20, Weight: 16},
	{ID: "ROLEX", Name: "劳力士", Rarity: rarityLow, Value: 20, Weight: 16},
	{ID: "SAYYID_POCKET_WATCH", Name: "赛义德的怀表", Rarity: rarityLow, Value: 20, Weight: 16},
	{ID: "ARCHIVED_EXPERIMENT_DATA", Name: "实验数据", Rarity: rarityLow, Value: 20, Weight: 16},
	{ID: "ULTRASONIC_CUTTER", Name: "超声波切割刀", Rarity: rarityLow, Value: 20, Weight: 16},
}

// CostForTimes 根据抽数返回总消耗；不支持的抽数返回错误。
//
// 当前仅支持 times == 1。
func CostForTimes(times int) (int, error) {
	if times == 1 {
		return SingleDrawCost, nil
	}
	return 0, errors.NewExposable(errors.CodeInvalidParam, "invalid lottery times: %d", times)
}

// DrawLottery 使用 crypto/rand 抽取 N 次结果。
//
// 行为：
//   - 当前仅支持 times == 1，返回长度为 1 的切片；其它值视为非法参数。
//   - 每抽中一件奖品，把奖品 Value 累加到 currentGuarantee；
//     若累加后 > MaxGuaranteedAmount，本次抽奖整体回滚：返回 INVALID_GUARANTEE
//     错误，不返回任何记录；调用方应据此拒绝整次 Intent（金币不扣、保底不变）。
//   - 成功时第二个返回值为最终保底金额（= 最后一条记录的 GuaranteeAfter）。
func DrawLottery(playerID string, times int, currentGuarantee int) ([]protocol.LotteryRecord, int, error) {
	cost, err := CostForTimes(times)
	if err != nil {
		return nil, currentGuarantee, err
	}
	now := time.Now().UnixMilli()
	out := make([]protocol.LotteryRecord, 0, times)
	guarantee := currentGuarantee
	for i := 0; i < times; i++ {
		item, err := rollOne(lotteryPool)
		if err != nil {
			return nil, currentGuarantee, errors.WithMessage(err, "roll lottery")
		}
		before := guarantee
		after := guarantee + item.Value
		if after > MaxGuaranteedAmount {
			// 整次抽奖回滚：保底不变、不下发记录。
			return nil, currentGuarantee, errors.NewExposable(errors.CodeInvalidParam, lotteryReasonInvalidGuarantee)
		}
		guarantee = after
		id, err := shortID()
		if err != nil {
			return nil, currentGuarantee, errors.WithMessage(err, "shortID")
		}
		out = append(out, protocol.LotteryRecord{
			ID:       id,
			PlayerID: playerID,
			Cost:     cost,
			Item: protocol.Item{
				ID:     item.ID,
				Name:   item.Name,
				Rarity: item.Rarity,
				Value:  item.Value,
			},
			GuaranteeDelta:  item.Value,
			GuaranteeBefore: before,
			GuaranteeAfter:  after,
			At:              now,
		})
	}
	return out, guarantee, nil
}

// rollOne 按权重 roll 一个物品；crypto/rand 提供随机源。
func rollOne(pool []LotteryItem) (LotteryItem, error) {
	if len(pool) == 0 {
		return LotteryItem{}, errors.New(errors.CodeInternal, "empty lottery pool")
	}
	var total float64
	for _, it := range pool {
		total += it.Weight
	}
	r, err := randFloat()
	if err != nil {
		return LotteryItem{}, err
	}
	target := r * total
	cum := 0.0
	for _, it := range pool {
		cum += it.Weight
		if target <= cum {
			return it, nil
		}
	}
	return pool[len(pool)-1], nil
}

// randFloat 返回 [0,1) 的均匀分布随机浮点数（crypto/rand）。
func randFloat() (float64, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0, err
	}
	// 取前 53 位（IEEE 754 双精度尾数），保证不超过 1。
	u := binary.BigEndian.Uint64(buf[:]) >> 11
	return float64(u) / (1 << 53), nil
}

// shortID 生成 10 字符 base64url 随机 ID。
func shortID() (string, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf[:])[:10], nil
}
