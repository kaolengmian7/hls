package room

import (
	"context"
	stdrand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"hls.dev/server/internal/game"
	"hls.dev/server/internal/protocol"
	"hls.dev/server/internal/repo"
	"hls.dev/server/pkg/errors"
)

// Intent 是落到 Room.inbox 的入站事件。
type Intent struct {
	Type    string
	Payload json.RawMessage
	ReqID   string
	From    *Client
}

// 内部 Intent 类型（不会从网络上接收）。
const (
	internalClientGone = "__CLIENT_GONE__"
)

// 各类超时常量。
//
// 本游戏定位为"小时级记账场景"：玩家通常每小时回来记一次账，期间应用会被切到后台。
// 因此宽限期统一拉长到 8h；定时巡检从 5s 放宽到 30s 以避免无谓 wakeup。
const (
	hostMigrationGrace = 8 * time.Hour
	playerDropGrace    = 8 * time.Hour
	idleRoomTimeout    = 8 * time.Hour
	tickInterval       = 30 * time.Second
	feedCapacity       = 50
	// maxNicknameRunes 昵称最大长度（按 Unicode 字符计），与前端 MAX_NICKNAME_LENGTH 对齐。
	maxNicknameRunes = 16
)

// playerState 是房间为每个玩家维护的运行时状态（不直接外泄到快照）。
type playerState struct {
	disconnectedAt time.Time
}

// TokenIssuer 抽象一个签发/校验 sessionToken 的能力。
// 真实实现见 pkg/token；Room 仅依赖接口便于测试。
type TokenIssuer interface {
	Sign(roomCode, playerID string) string
}

// Room 持有单个房间的权威状态及处理循环。
type Room struct {
	code     string
	hub      *Hub
	logger   logx.Logger
	tokens   TokenIssuer
	shopRepo *repo.ShopRepo   // nil = tests / no-persistence mode
	userIDs  map[string]int64 // playerID → auth user ID

	status      protocol.RoomStatus
	hostID      string
	players     map[string]*protocol.Player
	states      map[string]*playerState
	playerOrder []string
	clients     map[string]*Client

	guaranteedAmount int

	lotteryRecords    []protocol.LotteryRecord
	settlementRecords []protocol.SettlementRecord
	feed              []protocol.FeedEvent

	createdAt    time.Time
	lastActivity time.Time

	inbox    chan Intent
	regCh    chan registration
	doneCh   chan struct{}
	stopOnce bool
}

type registration struct {
	client        *Client
	playerID      string // 仅 reconnect 时携带；新客户端为空
	authenticated bool   // sessionToken 已验证
}

// newRoom 创建一个空房间，调用方在外部启动 Run goroutine。
//
// guaranteedAmount 来自 Hub.CreateRoom 已规范化的保底金额（单位 W），房间生命周期内不变。
func newRoom(hub *Hub, code string, hostNickname, hostAvatar, hostPlayerID string,
	hostUserID int64, guaranteedAmount int, shopRepo *repo.ShopRepo, logger logx.Logger, tokens TokenIssuer) *Room {
	now := time.Now()
	host := &protocol.Player{
		ID:        hostPlayerID,
		Nickname:  hostNickname,
		Avatar:    hostAvatar,
		Ready:     false,
		Coin:      0,
		IsHost:    true,
		Connected: false,
	}
	r := &Room{
		code:              code,
		hub:               hub,
		logger:            logger,
		tokens:            tokens,
		shopRepo:          shopRepo,
		userIDs:           map[string]int64{hostPlayerID: hostUserID},
		status:            protocol.RoomStatusWaiting,
		hostID:            hostPlayerID,
		players:           map[string]*protocol.Player{hostPlayerID: host},
		states:            map[string]*playerState{hostPlayerID: {disconnectedAt: now}},
		playerOrder:       []string{hostPlayerID},
		clients:           map[string]*Client{},
		guaranteedAmount:  guaranteedAmount,
		lotteryRecords:    []protocol.LotteryRecord{},
		settlementRecords: []protocol.SettlementRecord{},
		feed:              []protocol.FeedEvent{},
		createdAt:         now,
		lastActivity:      now,
		inbox:             make(chan Intent, 128),
		regCh:             make(chan registration, 8),
		doneCh:            make(chan struct{}),
	}
	return r
}

// dbWriteAsync fires a DB write in a background goroutine.
// Skipped when shopRepo is nil (test / no-persistence mode).
func (r *Room) dbWriteAsync(fn func() error) {
	if r.shopRepo == nil {
		return
	}
	go func() {
		if err := fn(); err != nil {
			r.logger.Errorw("shop db write", logx.Field("room", r.code), logx.Field("err", err))
		}
	}()
}

// GuaranteedAmount 返回该房间的保底金额（单位 W）。仅在 Run 之外的只读访问场景使用。
func (r *Room) GuaranteedAmount() int { return r.guaranteedAmount }

// Code 返回房间号。
func (r *Room) Code() string { return r.code }

// Status 返回当前房间状态（线程安全：仅在 loop 内修改，外部读取仅作为提示）。
func (r *Room) Status() protocol.RoomStatus { return r.status }

// PlayerCount 返回当前玩家数（同上：仅作提示）。
func (r *Room) PlayerCount() int { return len(r.players) }

// HasPlayer 返回该 playerID 当前是否仍占着房间内的座位（含断线未被 GC 的玩家）。
// 与 Status / PlayerCount 一致：仅在 Run 之外的只读提示场景使用，用于 REST 预检查
// "这是不是我还能回归的对局"。
func (r *Room) HasPlayer(playerID string) bool {
	if playerID == "" {
		return false
	}
	_, ok := r.players[playerID]
	return ok
}

// Submit 将 Intent 投递到 Room 处理循环。
// 非阻塞：满则记录错误并丢弃（避免业务侧背压）。
func (r *Room) Submit(it Intent) {
	select {
	case r.inbox <- it:
	default:
		r.logger.Errorw("room inbox full, drop intent",
			logx.Field("code", r.code),
			logx.Field("type", it.Type))
	}
}

// RegisterClient 由 handler 在 WebSocket 握手成功后调用：
//   - authenticated=true 表示 sessionToken 已通过校验，playerID 已知；按 reconnect 处理。
//   - authenticated=false 表示匿名新连接，等待客户端发 JOIN。
func (r *Room) RegisterClient(client *Client, playerID string, authenticated bool) {
	select {
	case r.regCh <- registration{client: client, playerID: playerID, authenticated: authenticated}:
	default:
		r.logger.Errorw("room reg full", logx.Field("code", r.code))
		client.SendEvent(protocol.EventJoinRejected, protocol.JoinRejectedPayload{
			Reason:  protocol.RejectOther,
			Message: "服务繁忙",
		}, "")
		client.Close()
	}
}

// OnClientGone 由 client.readPump 退出时调用，让 Room 感知断开。
func (r *Room) OnClientGone(client *Client) {
	r.Submit(Intent{Type: internalClientGone, From: client})
}

// Run 房间主循环。每个 Room 一个 goroutine。
func (r *Room) Run(ctx context.Context) {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	defer r.cleanup(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.doneCh:
			return
		case reg := <-r.regCh:
			r.handleRegistration(ctx, reg)
		case it := <-r.inbox:
			r.dispatch(ctx, it)
		case <-ticker.C:
			r.handleTick(ctx)
		}
	}
}

func (r *Room) stop() {
	if r.stopOnce {
		return
	}
	r.stopOnce = true
	close(r.doneCh)
}

func (r *Room) cleanup(ctx context.Context) {
	r.dbWriteAsync(func() error {
		return r.shopRepo.CloseRoom(context.Background(), r.code)
	})
	for _, c := range r.clients {
		if c == nil {
			continue
		}
		c.SendEvent(protocol.EventKick, protocol.KickPayload{Reason: protocol.KickDismissed}, "")
		c.Close()
	}
	r.clients = map[string]*Client{}
	r.status = protocol.RoomStatusClosed
	if r.hub != nil {
		r.hub.removeRoom(r.code)
	}
	logx.WithContext(ctx).Infow("room cleaned up",
		logx.Field("code", r.code))
}

/* ------------------------------ 注册 & dispatch ------------------------------ */

func (r *Room) handleRegistration(ctx context.Context, reg registration) {
	r.lastActivity = time.Now()
	if reg.authenticated && reg.playerID != "" {
		r.handleReconnect(ctx, reg.client, reg.playerID)
		return
	}
	// 匿名连接：等待客户端首帧 JOIN，readPump 已启动会自动转发。
	// 此处仅保留连接引用占位（无 playerID）。
}

func (r *Room) handleReconnect(ctx context.Context, c *Client, playerID string) {
	logger := logx.WithContext(ctx)
	player, ok := r.players[playerID]
	if !ok {
		logger.Infow("reconnect player not in room",
			logx.Field("code", r.code),
			logx.Field("playerId", playerID))
		c.SendEvent(protocol.EventJoinRejected, protocol.JoinRejectedPayload{
			Reason: protocol.RejectNotFound,
		}, "")
		c.Close()
		return
	}
	// 顶替旧连接
	if old, exist := r.clients[playerID]; exist && old != nil && old != c {
		old.SendEvent(protocol.EventKick, protocol.KickPayload{Reason: "REPLACED"}, "")
		old.Close()
	}
	c.playerID = playerID
	c.hasJoined = true
	c.tokenVerified = true
	r.clients[playerID] = c
	player.Connected = true
	if s, ok := r.states[playerID]; ok {
		s.disconnectedAt = time.Time{}
	}
	snap := r.buildSnapshot()
	c.SendEvent(protocol.EventJoinAccepted, protocol.JoinAcceptedPayload{
		Snapshot:     snap,
		SelfPlayerID: playerID,
		SessionToken: r.issueToken(playerID),
	}, "")
	r.broadcastSnapshot()
	logger.Infow("player reconnected",
		logx.Field("code", r.code),
		logx.Field("playerId", playerID))
}

func (r *Room) dispatch(ctx context.Context, it Intent) {
	r.lastActivity = time.Now()
	switch it.Type {
	case internalClientGone:
		r.handleClientGone(ctx, it.From)
		return
	case protocol.IntentJoin:
		r.handleJoin(ctx, it)
		return
	}
	// 其余 Intent 要求来源 client 已完成 JOIN。
	if it.From == nil || !it.From.hasJoined || it.From.playerID == "" {
		return
	}
	switch it.Type {
	case protocol.IntentLeave:
		r.handleLeave(ctx, it.From.playerID)
	case protocol.IntentDismissRoom:
		r.handleDismiss(ctx, it.From.playerID)
	case protocol.IntentChangeRole:
		r.handleChangeRole(ctx, it)
	case protocol.IntentChangeNickname:
		r.handleChangeNickname(ctx, it)
	case protocol.IntentReady:
		r.handleReady(ctx, it)
	case protocol.IntentStartGame:
		r.handleStartGame(ctx, it)
	case protocol.IntentBuyEquipment:
		r.handleBuyEquipment(ctx, it)
	case protocol.IntentDrawLottery:
		r.handleDrawLottery(ctx, it)
	case protocol.IntentSettleRound:
		r.handleSettleRound(ctx, it)
	case protocol.IntentCloseGame:
		r.handleCloseGame(ctx, it)
	case protocol.IntentPing:
		it.From.SendEvent(protocol.EventPong, map[string]any{}, it.ReqID)
	default:
		logx.WithContext(ctx).Errorw("unknown intent",
			logx.Field("code", r.code),
			logx.Field("type", it.Type))
		r.rejectAction(it.From, it.ReqID, "UNKNOWN_INTENT", "")
	}
}

/* ------------------------------ Intent 处理实现 ------------------------------ */

func (r *Room) handleJoin(ctx context.Context, it Intent) {
	logger := logx.WithContext(ctx)
	c := it.From
	if c == nil {
		return
	}
	if c.hasJoined {
		return
	}
	var p protocol.JoinPayload
	if err := json.Unmarshal(it.Payload, &p); err != nil {
		c.SendEvent(protocol.EventJoinRejected, protocol.JoinRejectedPayload{
			Reason:  protocol.RejectOther,
			Message: "invalid payload",
		}, it.ReqID)
		c.Close()
		return
	}
	if p.Nickname == "" {
		c.SendEvent(protocol.EventJoinRejected, protocol.JoinRejectedPayload{
			Reason:  protocol.RejectOther,
			Message: "nickname required",
		}, it.ReqID)
		c.Close()
		return
	}

	// 取连接 URL 上携带的 playerId（已在 handler 中存进 client.playerID 字段）。
	// 若 handler 未提供，则当前协议不允许匿名 JOIN，直接拒绝。
	playerID := c.playerID
	if playerID == "" {
		c.SendEvent(protocol.EventJoinRejected, protocol.JoinRejectedPayload{
			Reason:  protocol.RejectOther,
			Message: "playerId required",
		}, it.ReqID)
		c.Close()
		return
	}

	// 既有座位复用：playerID 命中且当前未连接 -> 直接回座。
	//
	// 这里**不**校验房间状态：玩家关浏览器后凭口令 + 本机 playerId 回归时，
	// 即便对局已经 PLAYING，只要其座位还在（未被 8h 宽限 GC），也应允许重新接入，
	// 以恢复其身份、角色与鼠币。这是 sessionToken 失效（清缓存 / 超 TTL）时的兜底路径；
	// 带有效 token 的重连仍走 handleReconnect。
	if existing, ok := r.players[playerID]; ok {
		// 已有相同 playerID
		if existing.Connected {
			c.SendEvent(protocol.EventJoinRejected, protocol.JoinRejectedPayload{
				Reason: protocol.RejectDup,
			}, it.ReqID)
			c.Close()
			return
		}
		// 复用：补充昵称/头像并标记上线
		existing.Nickname = p.Nickname
		if p.Avatar != "" {
			existing.Avatar = p.Avatar
		}
		existing.Connected = true
		if s, ok := r.states[playerID]; ok {
			s.disconnectedAt = time.Time{}
		}
		r.clients[playerID] = c
		c.hasJoined = true
		c.nickname = p.Nickname
		c.avatar = p.Avatar
		snap := r.buildSnapshot()
		c.SendEvent(protocol.EventJoinAccepted, protocol.JoinAcceptedPayload{
			Snapshot:     snap,
			SelfPlayerID: playerID,
			SessionToken: r.issueToken(playerID),
		}, it.ReqID)
		r.broadcastExcept(playerID, protocol.EventPlayerJoined, protocol.PlayerJoinedPayload{Player: *existing})
		r.broadcastSnapshot()
		logger.Infow("player joined (slot reuse)",
			logx.Field("code", r.code),
			logx.Field("playerId", playerID))
		return
	}

	// 全新玩家：对局已开始则不允许中途加入。
	if r.status != protocol.RoomStatusWaiting {
		c.SendEvent(protocol.EventJoinRejected, protocol.JoinRejectedPayload{
			Reason: protocol.RejectPlaying,
		}, it.ReqID)
		c.Close()
		return
	}

	if len(r.players) >= game.MaxPlayers {
		c.SendEvent(protocol.EventJoinRejected, protocol.JoinRejectedPayload{
			Reason: protocol.RejectFull,
		}, it.ReqID)
		c.Close()
		return
	}

	player := &protocol.Player{
		ID:        playerID,
		Nickname:  p.Nickname,
		Avatar:    p.Avatar,
		Ready:     false,
		Coin:      0,
		IsHost:    false,
		Connected: true,
	}
	r.players[playerID] = player
	r.states[playerID] = &playerState{}
	r.playerOrder = append(r.playerOrder, playerID)
	r.clients[playerID] = c
	c.hasJoined = true
	c.playerID = playerID
	c.nickname = p.Nickname
	c.avatar = p.Avatar
	r.userIDs[playerID] = c.userID
	uid := c.userID
	pid := playerID
	r.dbWriteAsync(func() error {
		return r.shopRepo.AddPlayer(context.Background(), r.code, uid, pid, false)
	})

	r.appendFeed("PLAYER_JOINED", map[string]any{
		"playerId": playerID,
		"nickname": p.Nickname,
	})
	snap := r.buildSnapshot()
	c.SendEvent(protocol.EventJoinAccepted, protocol.JoinAcceptedPayload{
		Snapshot:     snap,
		SelfPlayerID: playerID,
		SessionToken: r.issueToken(playerID),
	}, it.ReqID)
	r.broadcastExcept(playerID, protocol.EventPlayerJoined, protocol.PlayerJoinedPayload{Player: *player})
	r.broadcastSnapshot()
	logger.Infow("player joined",
		logx.Field("code", r.code),
		logx.Field("playerId", playerID))
}

func (r *Room) handleLeave(ctx context.Context, playerID string) {
	if playerID == r.hostID {
		// 房主主动离开 = 解散（也可改为迁移；当前实现解散）
		r.handleDismiss(ctx, playerID)
		return
	}
	r.removePlayer(ctx, playerID, false)
}

func (r *Room) handleDismiss(ctx context.Context, playerID string) {
	if playerID != r.hostID {
		// 非房主无权解散
		if c := r.clients[playerID]; c != nil {
			r.rejectAction(c, "", "FORBIDDEN", "仅房主可解散")
		}
		return
	}
	logx.WithContext(ctx).Infow("room dismissed",
		logx.Field("code", r.code),
		logx.Field("by", playerID))
	for pid, c := range r.clients {
		if c == nil {
			continue
		}
		c.SendEvent(protocol.EventKick, protocol.KickPayload{Reason: protocol.KickDismissed}, "")
		c.Close()
		_ = pid
	}
	r.stop()
}

func (r *Room) handleClientGone(ctx context.Context, c *Client) {
	if c == nil || c.playerID == "" {
		return
	}
	pid := c.playerID
	if cur, ok := r.clients[pid]; ok && cur == c {
		delete(r.clients, pid)
		if p, ok := r.players[pid]; ok {
			p.Connected = false
		}
		if s, ok := r.states[pid]; ok {
			s.disconnectedAt = time.Now()
		}
		logx.WithContext(ctx).Infow("client gone",
			logx.Field("code", r.code),
			logx.Field("playerId", pid))
		r.broadcastSnapshot()
	}
}

func (r *Room) handleChangeRole(ctx context.Context, it Intent) {
	if r.status != protocol.RoomStatusWaiting {
		r.rejectAction(it.From, it.ReqID, "INVALID_STATE", "")
		return
	}
	var p protocol.ChangeRolePayload
	if err := json.Unmarshal(it.Payload, &p); err != nil {
		r.rejectAction(it.From, it.ReqID, "INVALID_PARAM", "")
		return
	}
	if p.Role != protocol.RoleBoss && p.Role != protocol.RoleEscort {
		r.rejectAction(it.From, it.ReqID, "INVALID_PARAM", "")
		return
	}
	pid := it.From.playerID
	player, ok := r.players[pid]
	if !ok {
		return
	}
	if !game.CanAssignRole(toSliceCopy(r.players, r.playerOrder), p.Role, pid) {
		reason := "ROLE_TAKEN"
		if p.Role == protocol.RoleEscort {
			reason = "ROLE_FULL"
		}
		r.rejectAction(it.From, it.ReqID, reason, "该角色名额已满")
		return
	}
	player.Role = p.Role
	player.Ready = false
	pid2, role2 := pid, string(p.Role)
	r.dbWriteAsync(func() error {
		if pl, ok := r.players[pid2]; ok {
			return r.shopRepo.UpdatePlayerState(context.Background(), r.code, pid2, role2, pl.Coin, pl.Ready)
		}
		return nil
	})
	r.broadcast(protocol.EventRoleChanged, protocol.RoleChangedPayload{
		PlayerID: pid,
		Role:     p.Role,
	})
	r.broadcastSnapshot()
	logx.WithContext(ctx).Infow("role changed",
		logx.Field("code", r.code),
		logx.Field("playerId", pid),
		logx.Field("role", p.Role))
}

// handleChangeNickname 处理准备阶段的昵称修改（含可选名牌）。
//
// 行为：
//   - 仅在房间 WAITING 时允许；PLAYING / CLOSED 一律拒绝（INVALID_STATE）。
//   - 昵称去除首尾空白后不能为空；超出 maxNicknameRunes 时按字符截断。
//   - 仅更新该玩家的展示昵称，不影响角色 / 准备状态；随后广播快照对账。
func (r *Room) handleChangeNickname(ctx context.Context, it Intent) {
	if r.status != protocol.RoomStatusWaiting {
		r.rejectAction(it.From, it.ReqID, "INVALID_STATE", "")
		return
	}
	var p protocol.ChangeNicknamePayload
	if err := json.Unmarshal(it.Payload, &p); err != nil {
		r.rejectAction(it.From, it.ReqID, "INVALID_PARAM", "")
		return
	}
	nickname := strings.TrimSpace(p.Nickname)
	if nickname == "" {
		r.rejectAction(it.From, it.ReqID, "INVALID_PARAM", "昵称不能为空")
		return
	}
	if runes := []rune(nickname); len(runes) > maxNicknameRunes {
		nickname = string(runes[:maxNicknameRunes])
	}
	pid := it.From.playerID
	player, ok := r.players[pid]
	if !ok {
		return
	}
	if player.Nickname == nickname {
		return
	}
	player.Nickname = nickname
	if c := r.clients[pid]; c != nil {
		c.nickname = nickname
	}
	r.broadcastSnapshot()
	logx.WithContext(ctx).Infow("nickname changed",
		logx.Field("code", r.code),
		logx.Field("playerId", pid),
		logx.Field("nickname", nickname))
}

func (r *Room) handleReady(ctx context.Context, it Intent) {
	if r.status != protocol.RoomStatusWaiting {
		r.rejectAction(it.From, it.ReqID, "INVALID_STATE", "")
		return
	}
	var p protocol.ReadyPayload
	if err := json.Unmarshal(it.Payload, &p); err != nil {
		r.rejectAction(it.From, it.ReqID, "INVALID_PARAM", "")
		return
	}
	pid := it.From.playerID
	player, ok := r.players[pid]
	if !ok {
		return
	}
	if p.Ready && player.Role == "" {
		r.rejectAction(it.From, it.ReqID, "NO_ROLE", "请先选择角色")
		return
	}
	player.Ready = p.Ready
	pid2, ready2 := pid, p.Ready
	r.dbWriteAsync(func() error {
		if pl, ok := r.players[pid2]; ok {
			return r.shopRepo.UpdatePlayerState(context.Background(), r.code, pid2, string(pl.Role), pl.Coin, ready2)
		}
		return nil
	})
	r.broadcast(protocol.EventReadyChanged, protocol.ReadyChangedPayload{
		PlayerID: pid,
		Ready:    p.Ready,
	})
	r.broadcastSnapshot()
	logx.WithContext(ctx).Infow("ready changed",
		logx.Field("code", r.code),
		logx.Field("playerId", pid),
		logx.Field("ready", p.Ready))
}

func (r *Room) handleStartGame(ctx context.Context, it Intent) {
	if it.From.playerID != r.hostID {
		r.rejectAction(it.From, it.ReqID, "FORBIDDEN", "仅房主可开始游戏")
		return
	}
	if r.status != protocol.RoomStatusWaiting {
		r.rejectAction(it.From, it.ReqID, "INVALID_STATE", "")
		return
	}
	if err := game.CanStartGame(toSliceCopy(r.players, r.playerOrder)); err != nil {
		msg := err.Error()
		r.rejectAction(it.From, it.ReqID, "START_CONDITION", msg)
		return
	}
	for _, pid := range r.playerOrder {
		if p, ok := r.players[pid]; ok {
			p.Coin = game.InitialCoin(p.Role)
		}
	}
	r.status = protocol.RoomStatusPlaying
	orderSnap := append([]string(nil), r.playerOrder...)
	r.dbWriteAsync(func() error {
		bctx := context.Background()
		if err := r.shopRepo.UpdateRoomStatus(bctx, r.code, "playing"); err != nil {
			return err
		}
		for _, pid2 := range orderSnap {
			if pl, ok := r.players[pid2]; ok {
				if err := r.shopRepo.UpdatePlayerState(bctx, r.code, pid2, string(pl.Role), pl.Coin, pl.Ready); err != nil {
					return err
				}
			}
		}
		return nil
	})
	r.appendFeed("GAME_STARTED", map[string]any{})
	snap := r.buildSnapshot()
	r.broadcast(protocol.EventGameStarted, protocol.GameStartedPayload{Snapshot: snap})
	r.broadcastSnapshot()
	logx.WithContext(ctx).Infow("game started",
		logx.Field("code", r.code))
}

func (r *Room) handleBuyEquipment(ctx context.Context, it Intent) {
	if r.status != protocol.RoomStatusPlaying {
		r.rejectAction(it.From, it.ReqID, "INVALID_STATE", "")
		return
	}
	var p protocol.BuyEquipmentPayload
	if err := json.Unmarshal(it.Payload, &p); err != nil {
		r.rejectAction(it.From, it.ReqID, "INVALID_PARAM", "")
		return
	}
	pid := it.From.playerID
	player, ok := r.players[pid]
	if !ok {
		return
	}
	if player.Role != protocol.RoleEscort {
		r.rejectAction(it.From, it.ReqID, "WRONG_ROLE", "仅护航可购买装备")
		return
	}
	eq, err := game.ValidatePurchase(p.EquipmentID, p.Pick)
	if err != nil {
		reason := "INVALID_PARAM"
		if errors.CodeOf(err) == errors.CodeItemNotFound {
			reason = "NOT_FOUND"
		} else if msg := err.Error(); msg == "INVALID_PICK" || msg == "PICK_NOT_REQUIRED" {
			reason = msg
		}
		r.rejectAction(it.From, it.ReqID, reason, "")
		return
	}
	if player.Coin < eq.Price {
		r.rejectAction(it.From, it.ReqID, "NO_COIN", "金币不足")
		return
	}
	player.Coin -= eq.Price
	pid2, coin2 := pid, player.Coin
	r.dbWriteAsync(func() error {
		if pl, ok := r.players[pid2]; ok {
			return r.shopRepo.UpdatePlayerState(context.Background(), r.code, pid2, string(pl.Role), coin2, pl.Ready)
		}
		return nil
	})
	feedPayload := map[string]any{
		"playerId":    pid,
		"equipmentId": eq.ID,
	}
	if p.Pick != "" {
		feedPayload["pick"] = p.Pick
		if w := game.FindWeapon(p.Pick); w != nil {
			feedPayload["pickName"] = w.Name
		}
	}
	r.appendFeed("PURCHASE", feedPayload)
	r.broadcast(protocol.EventEquipmentPurchased, protocol.EquipmentPurchasedPayload{
		PlayerID:    pid,
		EquipmentID: eq.ID,
		Pick:        p.Pick,
	})
	r.broadcast(protocol.EventCoinChanged, protocol.CoinChangedPayload{
		Changes: []protocol.CoinChange{{
			PlayerID: pid, Delta: -eq.Price, Balance: player.Coin,
		}},
	})
	r.broadcastSnapshot()
	logx.WithContext(ctx).Infow("equipment purchased",
		logx.Field("code", r.code),
		logx.Field("playerId", pid),
		logx.Field("equipmentId", eq.ID),
		logx.Field("pick", p.Pick))
}

func (r *Room) handleDrawLottery(ctx context.Context, it Intent) {
	if r.status != protocol.RoomStatusPlaying {
		r.rejectAction(it.From, it.ReqID, "INVALID_STATE", "")
		return
	}
	var p protocol.DrawLotteryPayload
	if err := json.Unmarshal(it.Payload, &p); err != nil {
		r.rejectAction(it.From, it.ReqID, "INVALID_PARAM", "")
		return
	}
	pid := it.From.playerID
	player, ok := r.players[pid]
	if !ok {
		return
	}
	if player.Role != protocol.RoleBoss {
		r.rejectAction(it.From, it.ReqID, "WRONG_ROLE", "仅老板可抽奖")
		return
	}
	cost, err := game.CostForTimes(p.Times)
	if err != nil {
		r.rejectAction(it.From, it.ReqID, "INVALID_PARAM", err.Error())
		return
	}
	if player.Coin < cost {
		r.rejectAction(it.From, it.ReqID, "NO_COIN", "金币不足")
		return
	}
	// 抽奖随机 + 计算保底累加（纯函数）。若累加后越界，整次回滚：
	// 金币不扣、保底不变、不下发 LOTTERY_DRAWN。
	guaranteeBefore := r.guaranteedAmount
	records, newGuarantee, err := game.DrawLottery(pid, p.Times, guaranteeBefore)
	if err != nil {
		if errors.IsExposable(err) {
			r.rejectAction(it.From, it.ReqID, err.Error(), "")
		} else {
			r.rejectAction(it.From, it.ReqID, "INTERNAL", err.Error())
			logx.WithContext(ctx).Errorw("draw lottery",
				logx.Field("code", r.code),
				logx.Field("err", err))
		}
		return
	}

	// 原子状态更新（在 Room Loop 单 goroutine 内，无需加锁）：
	//  1. 扣金币；2. 更新保底；3. 记录抽奖流水 + Feed。
	player.Coin -= cost
	r.guaranteedAmount = newGuarantee
	pid2, coin2, guarantee2 := pid, player.Coin, newGuarantee
	lotteryRows := make([]repo.ShopLotteryRow, len(records))
	for i, rec := range records {
		lotteryRows[i] = repo.ShopLotteryRow{
			RoomCode:        r.code,
			PlayerID:        pid2,
			ItemID:          rec.Item.ID,
			ItemName:        rec.Item.Name,
			Rarity:          string(rec.Item.Rarity),
			ItemValue:       rec.Item.Value,
			GuaranteeBefore: rec.GuaranteeBefore,
			GuaranteeAfter:  rec.GuaranteeAfter,
		}
	}
	r.dbWriteAsync(func() error {
		bctx := context.Background()
		if pl, ok := r.players[pid2]; ok {
			if err := r.shopRepo.UpdatePlayerState(bctx, r.code, pid2, string(pl.Role), coin2, pl.Ready); err != nil {
				return err
			}
		}
		if err := r.shopRepo.UpdateGuaranteedAmount(bctx, r.code, guarantee2); err != nil {
			return err
		}
		return r.shopRepo.InsertLotteryRecords(bctx, lotteryRows)
	})
	// 头部插入，最新在前；与前端 UI 期望一致。
	r.lotteryRecords = append(append([]protocol.LotteryRecord{}, records...), r.lotteryRecords...)
	for _, rec := range records {
		r.appendFeed("LOTTERY", map[string]any{
			"playerId":        pid,
			"itemId":          rec.Item.ID,
			"itemName":        rec.Item.Name,
			"rarity":          rec.Item.Rarity,
			"itemValue":       rec.Item.Value,
			"guaranteeDelta":  rec.GuaranteeDelta,
			"guaranteeBefore": rec.GuaranteeBefore,
			"guaranteeAfter":  rec.GuaranteeAfter,
		})
	}

	// 发送顺序：
	//  1. 先给老板本人单播 LOTTERY_DRAWN，确保其先看到抽奖结果；
	//  2. 再向全员广播 STATE_SNAPSHOT，把公告 / feed 同步出去。
	// 与 docs/page.MD §3.2.2 「服务端裁决（抽奖 → 保底联动）」约定一致。
	r.broadcast(protocol.EventCoinChanged, protocol.CoinChangedPayload{
		Changes: []protocol.CoinChange{{
			PlayerID: pid, Delta: -cost, Balance: player.Coin,
		}},
	})
	if newGuarantee != guaranteeBefore {
		r.broadcast(protocol.EventGuaranteeChanged, protocol.GuaranteeChangedPayload{
			Before: guaranteeBefore,
			After:  newGuarantee,
		})
	}
	r.sendToPlayer(pid, protocol.EventLotteryDrawn, protocol.LotteryDrawnPayload{Records: records})
	r.broadcastSnapshot()
	logx.WithContext(ctx).Infow("lottery drawn",
		logx.Field("code", r.code),
		logx.Field("playerId", pid),
		logx.Field("times", p.Times),
		logx.Field("guaranteeBefore", guaranteeBefore),
		logx.Field("guaranteeAfter", newGuarantee))
}

// handleSettleRound 处理房主发起的本局结算。
//
// 行为：
//  1. 仅房主、且房间处于 PLAYING 时允许；否则回 ACTION_REJECTED。
//  2. 通过 game.ComputeSettlement 完成所有业务校验与金币计算（纯函数）。
//  3. 在 Room Loop 同一 goroutine 内原子地：写入新保底金额、累加每位玩家的 coin、
//     追加一条 SettlementRecord、追加一条 SETTLEMENT FeedEvent。
//  4. 顺序广播：GUARANTEE_CHANGED（仅当 delta != 0）→ COIN_CHANGED → ROUND_SETTLED →
//     STATE_SNAPSHOT，保证客户端按这个顺序对账。
//
// 设计要点：
//   - 房间状态保持 PLAYING，不会因结算自动结束；房主可连续多次结算直至主动 DISMISS_ROOM。
//   - 失败时不会修改任何状态，错误 reason 通过 ACTION_REJECTED 透传给客户端。
func (r *Room) handleSettleRound(ctx context.Context, it Intent) {
	if it.From.playerID != r.hostID {
		r.rejectAction(it.From, it.ReqID, "FORBIDDEN", "仅房主可结算本局")
		return
	}
	if r.status != protocol.RoomStatusPlaying {
		r.rejectAction(it.From, it.ReqID, "INVALID_STATE", "")
		return
	}
	var p protocol.SettleRoundPayload
	if err := json.Unmarshal(it.Payload, &p); err != nil {
		r.rejectAction(it.From, it.ReqID, "INVALID_PARAMS", "")
		return
	}

	roleByID := make(map[string]protocol.Role, len(r.players))
	for pid, pl := range r.players {
		roleByID[pid] = pl.Role
	}
	result, err := game.ComputeSettlement(
		p.Outcome, p.GuaranteeDelta, r.guaranteedAmount,
		p.PerPlayer, append([]string(nil), r.playerOrder...), roleByID,
	)
	if err != nil {
		// game 包通过 .Error() 返回 INVALID_PARAMS / INVALID_GUARANTEE。
		r.rejectAction(it.From, it.ReqID, err.Error(), "")
		return
	}

	// 原子状态更新（在 Room Loop 单 goroutine 内，无需加锁）。
	r.guaranteedAmount = result.GuaranteeAfter
	coinChanges := make([]protocol.CoinChange, 0, len(result.PerPlayer))
	for _, pp := range result.PerPlayer {
		player, ok := r.players[pp.PlayerID]
		if !ok {
			// 不应发生：ComputeSettlement 已经做过完备性校验。
			continue
		}
		player.Coin += pp.CoinDelta
		coinChanges = append(coinChanges, protocol.CoinChange{
			PlayerID: pp.PlayerID,
			Delta:    pp.CoinDelta,
			Balance:  player.Coin,
		})
	}

	recID, idErr := shortEventID()
	if idErr != nil {
		recID = "settle"
	}
	record := protocol.SettlementRecord{
		ID:              recID,
		At:              time.Now().UnixMilli(),
		Outcome:         result.Outcome,
		GuaranteeDelta:  result.GuaranteeDelta,
		GuaranteeBefore: result.GuaranteeBefore,
		GuaranteeAfter:  result.GuaranteeAfter,
		PerPlayer:       result.PerPlayer,
	}
	// 头部插入，最新在前；与抽奖记录一致，便于前端按时间倒序展示。
	r.settlementRecords = append([]protocol.SettlementRecord{record}, r.settlementRecords...)

	guarantee2 := result.GuaranteeAfter
	perPlayerJSON, _ := json.Marshal(result.PerPlayer)
	settleRow := repo.ShopSettlementRow{
		ID:              record.ID,
		RoomCode:        r.code,
		Outcome:         string(result.Outcome),
		GuaranteeDelta:  result.GuaranteeDelta,
		GuaranteeBefore: result.GuaranteeBefore,
		GuaranteeAfter:  result.GuaranteeAfter,
		PerPlayer:       string(perPlayerJSON),
	}
	r.dbWriteAsync(func() error {
		bctx := context.Background()
		if err := r.shopRepo.UpdateGuaranteedAmount(bctx, r.code, guarantee2); err != nil {
			return err
		}
		for _, pp := range result.PerPlayer {
			if pl, ok := r.players[pp.PlayerID]; ok {
				if err := r.shopRepo.UpdatePlayerState(bctx, r.code, pp.PlayerID, string(pl.Role), pl.Coin, pl.Ready); err != nil {
					return err
				}
			}
		}
		return r.shopRepo.InsertSettlement(bctx, settleRow)
	})

	r.appendFeed("SETTLEMENT", map[string]any{
		"outcome":         string(result.Outcome),
		"guaranteeDelta":  result.GuaranteeDelta,
		"guaranteeBefore": result.GuaranteeBefore,
		"guaranteeAfter":  result.GuaranteeAfter,
	})

	if result.GuaranteeDelta != 0 {
		r.broadcast(protocol.EventGuaranteeChanged, protocol.GuaranteeChangedPayload{
			Before: result.GuaranteeBefore,
			After:  result.GuaranteeAfter,
		})
	}
	if len(coinChanges) > 0 {
		r.broadcast(protocol.EventCoinChanged, protocol.CoinChangedPayload{Changes: coinChanges})
	}
	r.broadcast(protocol.EventRoundSettled, protocol.RoundSettledPayload{Record: record})
	r.broadcastSnapshot()

	logx.WithContext(ctx).Infow("round settled",
		logx.Field("code", r.code),
		logx.Field("by", it.From.playerID),
		logx.Field("outcome", string(result.Outcome)),
		logx.Field("guaranteeBefore", result.GuaranteeBefore),
		logx.Field("guaranteeAfter", result.GuaranteeAfter))
}

// handleCloseGame 处理房主结单：强制结束游戏，断开所有玩家。
//
// 行为：
//  1. 仅房主、且房间处于 PLAYING 时允许；否则回 ACTION_REJECTED。
//  2. 向全员广播 KICK 事件（Reason: protocol.KickClosed），关闭所有客户端连接。
//  3. 调用 r.stop() 触发房间关闭流程，cleanup() 会写入 CloseRoom DB 记录。
func (r *Room) handleCloseGame(ctx context.Context, it Intent) {
	if it.From.playerID != r.hostID {
		r.rejectAction(it.From, it.ReqID, "FORBIDDEN", "仅房主可结单")
		return
	}
	if r.status != protocol.RoomStatusPlaying {
		r.rejectAction(it.From, it.ReqID, "INVALID_STATE", "结单仅在游戏进行中有效")
		return
	}
	logx.WithContext(ctx).Infow("close game", logx.Field("code", r.code), logx.Field("by", it.From.playerID))
	for _, c := range r.clients {
		if c == nil {
			continue
		}
		c.SendEvent(protocol.EventKick, protocol.KickPayload{Reason: protocol.KickClosed}, "")
		c.Close()
	}
	r.stop() // triggers cleanup() → DB CloseRoom
}

/* ------------------------------ 定时巡检 ------------------------------ */

func (r *Room) handleTick(ctx context.Context) {
	now := time.Now()

	// 房主迁移：若房主断线超过 30s 且仍有其他在线玩家，迁移。
	if hostState, ok := r.states[r.hostID]; ok && !hostState.disconnectedAt.IsZero() {
		if now.Sub(hostState.disconnectedAt) >= hostMigrationGrace {
			r.migrateHost(ctx)
		}
	}

	// 非房主断线超时清除
	stalePlayers := make([]string, 0)
	for pid, s := range r.states {
		if pid == r.hostID {
			continue
		}
		if !s.disconnectedAt.IsZero() && now.Sub(s.disconnectedAt) >= playerDropGrace {
			stalePlayers = append(stalePlayers, pid)
		}
	}
	for _, pid := range stalePlayers {
		r.removePlayer(ctx, pid, true)
	}

	// 空房 GC
	if r.hasNoConnectedClients() && now.Sub(r.lastActivity) >= idleRoomTimeout {
		logx.WithContext(ctx).Infow("room idle, GC",
			logx.Field("code", r.code))
		r.stop()
	}
}

func (r *Room) hasNoConnectedClients() bool {
	for _, p := range r.players {
		if p.Connected {
			return false
		}
	}
	return true
}

func (r *Room) migrateHost(ctx context.Context) {
	for _, pid := range r.playerOrder {
		if pid == r.hostID {
			continue
		}
		if p, ok := r.players[pid]; ok && p.Connected {
			oldHost := r.hostID
			r.hostID = pid
			if oldP, ok := r.players[oldHost]; ok {
				oldP.IsHost = false
			}
			p.IsHost = true
			r.broadcast(protocol.EventHostChanged, protocol.HostChangedPayload{NewHostID: pid})
			r.broadcastSnapshot()
			logx.WithContext(ctx).Infow("host migrated",
				logx.Field("code", r.code),
				logx.Field("from", oldHost),
				logx.Field("to", pid))
			return
		}
	}
	// 没有可用继任者
	logx.WithContext(ctx).Infow("no host successor, dismiss room",
		logx.Field("code", r.code))
	r.stop()
}

func (r *Room) removePlayer(ctx context.Context, pid string, broadcastPlayerLeft bool) {
	if _, ok := r.players[pid]; !ok {
		return
	}
	delete(r.players, pid)
	delete(r.states, pid)
	for i, id := range r.playerOrder {
		if id == pid {
			r.playerOrder = append(r.playerOrder[:i], r.playerOrder[i+1:]...)
			break
		}
	}
	if c := r.clients[pid]; c != nil {
		c.Close()
		delete(r.clients, pid)
	}
	rpid := pid
	r.dbWriteAsync(func() error {
		return r.shopRepo.RemovePlayer(context.Background(), r.code, rpid)
	})
	if broadcastPlayerLeft {
		r.appendFeed("PLAYER_LEFT", map[string]any{"playerId": pid})
		r.broadcast(protocol.EventPlayerLeft, protocol.PlayerLeftPayload{PlayerID: pid})
	}

	if pid == r.hostID {
		r.migrateHost(ctx)
	}
	r.broadcastSnapshot()
}

/* ------------------------------ 广播工具 ------------------------------ */

func (r *Room) broadcast(eventType string, payload any) {
	for _, c := range r.clients {
		if c == nil {
			continue
		}
		c.SendEvent(eventType, payload, "")
	}
}

func (r *Room) sendToPlayer(playerID, eventType string, payload any) {
	c, ok := r.clients[playerID]
	if !ok || c == nil {
		return
	}
	c.SendEvent(eventType, payload, "")
}

func (r *Room) broadcastExcept(exceptPlayerID, eventType string, payload any) {
	for pid, c := range r.clients {
		if c == nil || pid == exceptPlayerID {
			continue
		}
		c.SendEvent(eventType, payload, "")
	}
}

func (r *Room) broadcastSnapshot() {
	snap := r.buildSnapshot()
	r.broadcast(protocol.EventStateSnapshot, protocol.StateSnapshotPayload{Snapshot: snap})
}

func (r *Room) rejectAction(c *Client, reqID, reason, message string) {
	if c == nil {
		return
	}
	c.SendEvent(protocol.EventActionRejected, protocol.ActionRejectedPayload{
		ReqID:   reqID,
		Reason:  reason,
		Message: message,
	}, reqID)
}

func (r *Room) appendFeed(kind string, payload map[string]any) {
	id, err := shortEventID()
	if err != nil {
		id = "evt"
	}
	ev := protocol.FeedEvent{
		ID:      id,
		At:      time.Now().UnixMilli(),
		Kind:    kind,
		Payload: payload,
	}
	r.feed = append([]protocol.FeedEvent{ev}, r.feed...)
	if len(r.feed) > feedCapacity {
		r.feed = r.feed[:feedCapacity]
	}
}

/* ------------------------------ 私有工具 ------------------------------ */

func toSliceCopy(m map[string]*protocol.Player, order []string) []protocol.Player {
	out := make([]protocol.Player, 0, len(m))
	for _, id := range order {
		if p, ok := m[id]; ok {
			out = append(out, *p)
		}
	}
	return out
}

func shortEventID() (string, error) {
	var buf [8]byte
	if _, err := stdrand.Read(buf[:]); err != nil {
		return "", errors.WithMessage(err, "rand")
	}
	return base64.RawURLEncoding.EncodeToString(buf[:])[:10], nil
}

// issueToken 通过注入的 TokenIssuer 为玩家签发 sessionToken。
func (r *Room) issueToken(playerID string) string {
	if r.tokens == nil {
		return ""
	}
	return r.tokens.Sign(r.code, playerID)
}
