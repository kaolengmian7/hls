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
	ID       string
	Name     string
	Rarity   string
	Value    int
	Weight   float64
	ImageURL string
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
// 共 51 件奖品，价值合计 10348W。每件奖品的 Value 在抽中时累加到房间
// GuaranteedAmount；如累加后 newGuarantee > MaxGuaranteedAmount，
// 则整次抽奖回滚（金币不扣、保底不变、不下发 LOTTERY_DRAWN）。
var lotteryPool = []LotteryItem{
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

// GetPool 返回奖品池的客户端视图（不含 Weight）。
func GetPool() []protocol.Item {
	items := make([]protocol.Item, len(lotteryPool))
	for i, it := range lotteryPool {
		items[i] = protocol.Item{
			ID:       it.ID,
			Name:     it.Name,
			Rarity:   it.Rarity,
			Value:    it.Value,
			ImageURL: it.ImageURL,
		}
	}
	return items
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
// highBoost is the weight multiplier for HIGH-rarity items. Pass 1.0 for no
// boost. Values > 1 increase the probability of HIGH items proportionally.
func DrawLottery(playerID string, times int, currentGuarantee int, highBoost float64) ([]protocol.LotteryRecord, int, error) {
	cost, err := CostForTimes(times)
	if err != nil {
		return nil, currentGuarantee, err
	}
	now := time.Now().UnixMilli()
	out := make([]protocol.LotteryRecord, 0, times)
	guarantee := currentGuarantee
	for i := 0; i < times; i++ {
		pool := lotteryPool
		if highBoost != 1.0 {
			pool = applyHighBoost(lotteryPool, highBoost)
		}
		item, err := rollOne(pool)
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
				ID:       item.ID,
				Name:     item.Name,
				Rarity:   item.Rarity,
				Value:    item.Value,
				ImageURL: item.ImageURL,
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

// applyHighBoost returns a new pool slice with HIGH item weights multiplied by
// factor. MEDIUM and LOW items are unchanged. The original slice is not mutated.
func applyHighBoost(src []LotteryItem, factor float64) []LotteryItem {
	out := make([]LotteryItem, len(src))
	copy(out, src)
	for i := range out {
		if out[i].Rarity == rarityHigh {
			out[i].Weight *= factor
		}
	}
	return out
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
