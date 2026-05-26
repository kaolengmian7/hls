package room

import (
	"context"
	"sync"

	"github.com/zeromicro/go-zero/core/logx"

	"hls.dev/server/internal/game"
	"hls.dev/server/internal/protocol"
	"hls.dev/server/pkg/code"
	"hls.dev/server/pkg/errors"
)

// Hub 全局房间表。线程安全。
//
// 仅做三件事：
//   1. 创建房间（分配 Code、启动 Room.Run goroutine）。
//   2. 按 Code 查找房间，转交连接。
//   3. 房间生命结束后释放 Code。
type Hub struct {
	ctx       context.Context
	generator *code.Generator
	tokens    TokenIssuer
	logger    logx.Logger

	mu    sync.RWMutex
	rooms map[string]*Room
}

// NewHub 创建 Hub。
func NewHub(ctx context.Context, gen *code.Generator, tokens TokenIssuer, logger logx.Logger) *Hub {
	return &Hub{
		ctx:       ctx,
		generator: gen,
		tokens:    tokens,
		logger:    logger,
		rooms:     map[string]*Room{},
	}
}

// CreateRoom 分配一个新房间并预占房主玩家。
// 返回 room、房主 sessionToken。
//
// guaranteedAmount 是房间级"保底金额"（单位 W），传 0 表示采用 game.DefaultGuaranteedAmount；
// 越界则返回可暴露错误。
func (h *Hub) CreateRoom(nickname, avatar, hostPlayerID string, guaranteedAmount int) (*Room, string, error) {
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
	r := newRoom(h, c, nickname, avatar, hostPlayerID, normalizedGuarantee, h.logger, h.tokens)

	h.mu.Lock()
	h.rooms[c] = r
	h.mu.Unlock()

	go r.Run(h.ctx)

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
type RoomSummary struct {
	Exists      bool                `json:"exists"`
	Status      protocol.RoomStatus `json:"status,omitempty"`
	PlayerCount int                 `json:"playerCount,omitempty"`
}

// QueryRoom 用于 REST 预检查。
func (h *Hub) QueryRoom(c string) RoomSummary {
	r := h.GetRoom(c)
	if r == nil {
		return RoomSummary{Exists: false}
	}
	return RoomSummary{
		Exists:      true,
		Status:      r.Status(),
		PlayerCount: r.PlayerCount(),
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
