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

// lotteryPool 由 InitPool 在服务启动时从配置注入；零值表示未初始化。
var lotteryPool []LotteryItem

// InitPool 在服务启动时由 svc.NewServiceContext 调用，写入奖品池。
// 池为空时返回错误；调用方应 logx.Must，使进程立即退出。
// InitPool 对 items 做防御性拷贝，外部修改不影响已注入的池。
func InitPool(items []LotteryItem) error {
	if len(items) == 0 {
		return errors.New(errors.CodeInternal,
			"lottery pool is empty: configure Shop.LotteryPool in gameserver.yaml")
	}
	lotteryPool = append([]LotteryItem(nil), items...)
	return nil
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
//   - highBoost 是 HIGH 物品的权重倍数；传 1.0 表示无提升。
func DrawLottery(playerID string, times int, currentGuarantee int, highBoost float64) ([]protocol.LotteryRecord, int, error) {
	cost, err := CostForTimes(times)
	if err != nil {
		return nil, currentGuarantee, err
	}
	now := time.Now().UnixMilli()
	out := make([]protocol.LotteryRecord, 0, times)
	guarantee := currentGuarantee
	pool := lotteryPool
	if highBoost != 1.0 {
		pool = applyHighBoost(lotteryPool, highBoost)
	}
	for i := 0; i < times; i++ {
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
