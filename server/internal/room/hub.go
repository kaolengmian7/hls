package room

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"hls.dev/server/internal/game"
	"hls.dev/server/internal/protocol"
	"hls.dev/server/internal/repo"
	"hls.dev/server/pkg/code"
	"hls.dev/server/pkg/errors"
)

// Hub 全局房间表。线程安全。
//
// 仅做三件事：
//  1. 创建房间（分配 Code、启动 Room.Run goroutine）。
//  2. 按 Code 查找房间，转交连接。
//  3. 房间生命结束后释放 Code。
type Hub struct {
	ctx          context.Context
	generator    *code.Generator
	tokens       TokenIssuer
	shopRepo     *repo.ShopRepo
	logger       logx.Logger
	vipUsers     map[int64]struct{}
	vipHighBoost float64

	mu    sync.RWMutex
	rooms map[string]*Room
}

// NewHub 创建 Hub。vipUsers 是专属高爆率账号集合；vipHighBoost 是 HIGH 物品权重倍数。
func NewHub(ctx context.Context, gen *code.Generator, tokens TokenIssuer, shopRepo *repo.ShopRepo, logger logx.Logger, vipUserIDs []int64, vipHighBoost float64) *Hub {
	vipSet := make(map[int64]struct{}, len(vipUserIDs))
	for _, uid := range vipUserIDs {
		vipSet[uid] = struct{}{}
	}
	if vipHighBoost <= 0 {
		vipHighBoost = 1.0
	}
	return &Hub{
		ctx:          ctx,
		generator:    gen,
		tokens:       tokens,
		shopRepo:     shopRepo,
		logger:       logger,
		vipUsers:     vipSet,
		vipHighBoost: vipHighBoost,
		rooms:        map[string]*Room{},
	}
}

// CreateRoom 分配一个新房间并预占房主玩家。
// 返回 room、房主 sessionToken。
//
// guaranteedAmount 是房间级"保底金额"（单位 W），传 0 表示采用 game.DefaultGuaranteedAmount；
// 越界则返回可暴露错误。
func (h *Hub) CreateRoom(nickname, avatar, hostPlayerID string, hostUserID int64, guaranteedAmount int) (*Room, string, error) {
	if nickname == "" {
		return nil, "", errors.NewExposable(errors.CodeInvalidParam, "nickname required")
	}
	if hostPlayerID == "" {
		return nil, "", errors.NewExposable(errors.CodeInvalidParam, "hostPlayerId required")
	}
	normalizedGuarantee, err := game.NormalizeGuaranteedAmount(guaranteedAmount)
	if err != nil {
		return nil, "", err
	}

	c, err := h.generator.Allocate()
	if err != nil {
		return nil, "", errors.WithMessage(err, "allocate code")
	}
	r := newRoom(h, c, nickname, avatar, hostPlayerID, hostUserID, normalizedGuarantee, h.shopRepo, h.logger, h.tokens)

	h.mu.Lock()
	h.rooms[c] = r
	h.mu.Unlock()

	go r.Run(h.ctx)

	if h.shopRepo != nil {
		hpid := hostPlayerID
		huid := hostUserID
		ga := normalizedGuarantee
		roomCode := c
		go func() {
			bctx := context.Background()
			if err := h.shopRepo.CreateRoom(bctx, roomCode, hpid, ga); err != nil {
				h.logger.Errorw("db create room", logx.Field("err", err))
				return
			}
			if err := h.shopRepo.AddPlayer(bctx, roomCode, huid, hpid, true); err != nil {
				h.logger.Errorw("db add host player", logx.Field("err", err))
			}
		}()
	}

	logx.WithContext(h.ctx).Infow("room created",
		logx.Field("code", c),
		logx.Field("hostId", hostPlayerID),
		logx.Field("guaranteedAmount", normalizedGuarantee))

	tok := ""
	if h.tokens != nil {
		tok = h.tokens.Sign(c, hostPlayerID)
	}
	return r, tok, nil
}

// GetRoom 按 Code 查找房间；不存在或已关闭返回 nil。
func (h *Hub) GetRoom(c string) *Room {
	h.mu.RLock()
	defer h.mu.RUnlock()
	r, ok := h.rooms[c]
	if !ok {
		return nil
	}
	return r
}

// RoomSummary 是给 REST 接口的简要状态。
//
// IsMember 表示请求方携带的 playerId 当前是否仍占着该房间的座位（含断线未被 GC 者）。
// 前端据此决定走"回归（恢复身份/鼠币）"还是"全新加入"：回归不受 PLAYING / 满员限制。
type RoomSummary struct {
	Exists      bool                `json:"exists"`
	Status      protocol.RoomStatus `json:"status,omitempty"`
	PlayerCount int                 `json:"playerCount,omitempty"`
	IsMember    bool                `json:"isMember,omitempty"`
}

// QueryRoom 用于 REST 预检查。
//
// playerID 可为空；非空时回填 IsMember，供前端判断是否是"回归本局"的老玩家。
func (h *Hub) QueryRoom(c, playerID string) RoomSummary {
	r := h.GetRoom(c)
	if r == nil {
		return RoomSummary{Exists: false}
	}
	return RoomSummary{
		Exists:      true,
		Status:      r.Status(),
		PlayerCount: r.PlayerCount(),
		IsMember:    r.HasPlayer(playerID),
	}
}

// removeRoom 由 Room.cleanup 回调，清理引用并释放 Code。
func (h *Hub) removeRoom(c string) {
	h.mu.Lock()
	delete(h.rooms, c)
	h.mu.Unlock()
	h.generator.Release(c)
	logx.WithContext(h.ctx).Infow("room removed",
		logx.Field("code", c))
}

// Stats 返回当前房间数等指标，便于诊断。
func (h *Hub) Stats() (rooms int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms)
}

// RestoreFromDB rebuilds in-memory rooms from active DB rows at startup.
func (h *Hub) RestoreFromDB(ctx context.Context) error {
	if h.shopRepo == nil {
		return nil
	}
	roomRows, err := h.shopRepo.LoadActiveRooms(ctx)
	if err != nil {
		return err
	}
	for _, rr := range roomRows {
		players, err := h.shopRepo.LoadRoomPlayers(ctx, rr.Code)
		if err != nil {
			h.logger.Errorw("restore players", logx.Field("code", rr.Code), logx.Field("err", err))
			continue
		}
		if len(players) == 0 {
			continue
		}
		lotteryRows, _ := h.shopRepo.LoadLotteryRecords(ctx, rr.Code)
		settlementRows, _ := h.shopRepo.LoadSettlements(ctx, rr.Code)

		rm := h.buildRestoredRoom(rr, players, lotteryRows, settlementRows)
		h.mu.Lock()
		h.rooms[rr.Code] = rm
		h.mu.Unlock()
		h.generator.Reserve(rr.Code)
		go rm.Run(h.ctx)
		h.logger.Infow("room restored", logx.Field("code", rr.Code), logx.Field("players", len(players)))
	}
	return nil
}

func (h *Hub) buildRestoredRoom(
	rr repo.ShopRoomRow,
	players []repo.ShopRoomPlayerRow,
	lotteryRows []repo.ShopLotteryRow,
	settlementRows []repo.ShopSettlementRow,
) *Room {
	now := time.Now()
	rm := &Room{
		code:              rr.Code,
		hub:               h,
		logger:            h.logger,
		tokens:            h.tokens,
		shopRepo:          h.shopRepo,
		status:            protocol.RoomStatus(rr.Status),
		hostID:            rr.HostPlayerID,
		players:           make(map[string]*protocol.Player),
		states:            make(map[string]*playerState),
		playerOrder:       []string{},
		clients:           make(map[string]*Client),
		userIDs:           make(map[string]int64),
		guaranteedAmount:  rr.GuaranteedAmount,
		lotteryRecords:    make([]protocol.LotteryRecord, 0, len(lotteryRows)),
		settlementRecords: make([]protocol.SettlementRecord, 0, len(settlementRows)),
		feed:              make([]protocol.FeedEvent, 0),
		createdAt:         rr.CreatedAt,
		lastActivity:      now,
		inbox:             make(chan Intent, 128),
		regCh:             make(chan registration, 8),
		doneCh:            make(chan struct{}),
	}
	for _, pr := range players {
		rm.players[pr.PlayerID] = &protocol.Player{
			ID:        pr.PlayerID,
			Nickname:  pr.PlayerID, // placeholder; refreshed on reconnect via JOIN
			Ready:     pr.Ready,
			Coin:      pr.Coin,
			IsHost:    pr.IsHost,
			Connected: false,
			Role:      protocol.Role(pr.Role),
		}
		rm.states[pr.PlayerID] = &playerState{disconnectedAt: now}
		rm.playerOrder = append(rm.playerOrder, pr.PlayerID)
		rm.userIDs[pr.PlayerID] = pr.UserID
	}
	for _, lr := range lotteryRows {
		rm.lotteryRecords = append(rm.lotteryRecords, protocol.LotteryRecord{
			PlayerID: lr.PlayerID,
			At:       lr.CreatedAt.UnixMilli(),
			Item: protocol.Item{
				ID:     lr.ItemID,
				Name:   lr.ItemName,
				Rarity: lr.Rarity,
				Value:  lr.ItemValue,
			},
			GuaranteeDelta:  lr.GuaranteeAfter - lr.GuaranteeBefore,
			GuaranteeBefore: lr.GuaranteeBefore,
			GuaranteeAfter:  lr.GuaranteeAfter,
		})
	}
	for _, sr := range settlementRows {
		var perPlayer []protocol.SettlementPerPlayer
		_ = json.Unmarshal([]byte(sr.PerPlayer), &perPlayer)
		rm.settlementRecords = append(rm.settlementRecords, protocol.SettlementRecord{
			ID:              sr.ID,
			At:              sr.CreatedAt.UnixMilli(),
			Outcome:         protocol.SettlementOutcome(sr.Outcome),
			GuaranteeDelta:  sr.GuaranteeDelta,
			GuaranteeBefore: sr.GuaranteeBefore,
			GuaranteeAfter:  sr.GuaranteeAfter,
			PerPlayer:       perPlayer,
		})
	}
	return rm
}
