package room

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"

	"hls.dev/server/internal/protocol"
)

// Client 表示一条 WebSocket 长连接 + 其上挂载的玩家信息。
//
// readPump / writePump 各运行在独立 goroutine；
// 业务侧只通过 Send([]byte) / Close() 与之交互。
type Client struct {
	conn      *websocket.Conn
	room      *Room
	playerID  string
	hasJoined bool
	nickname  string
	avatar    string
	tokenVerified bool

	send   chan []byte
	closed chan struct{}

	mu       sync.Mutex
	isClosed bool

	logger logx.Logger
}

// 写超时 / 读超时 / 心跳超时常量。
//
// 设备休眠或浏览器后台 throttle 时无法及时回 Pong，过短的 pongWait 会把
// "短暂离线但很快会回来"的玩家误判为掉线。这里取 3 分钟作为单条 WS 连接的存活上限：
// 真的回不来时由前端 visibilitychange 触发主动重连，由 sessionToken 完成续接。
const (
	writeWait      = 10 * time.Second
	pongWait       = 3 * time.Minute
	pingPeriod     = (pongWait * 9) / 10 // 服务端发 ping
	maxMessageSize = 32 * 1024
	sendQueueSize  = 64
)

// NewClient 创建 client；conn 必须已完成 WebSocket Upgrade。
// playerID 来自 WebSocket query 参数；JOIN 之前作为预绑定身份，JOIN 时与首帧 nickname 配对入场。
func NewClient(conn *websocket.Conn, room *Room, playerID string, logger logx.Logger) *Client {
	return &Client{
		conn:     conn,
		room:     room,
		playerID: playerID,
		send:     make(chan []byte, sendQueueSize),
		closed:   make(chan struct{}),
		logger:   logger,
	}
}

// PlayerID 返回该连接绑定的玩家 ID（JOIN 成功前为空）。
func (c *Client) PlayerID() string { return c.playerID }

// HasJoined 是否已完成 JOIN。
func (c *Client) HasJoined() bool { return c.hasJoined }

// Send 入队一条消息；若 send chan 已满则视为慢客户端，强制关闭连接。
func (c *Client) Send(data []byte) {
	c.mu.Lock()
	if c.isClosed {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
	select {
	case c.send <- data:
	default:
		c.logger.Errorw("client send queue full, closing",
			logx.Field("playerId", c.playerID))
		c.Close()
	}
}

// SendEvent 序列化并发送一条服务端事件。
func (c *Client) SendEvent(eventType string, payload any, reqID string) {
	out := protocol.Outgoing{
		Type:    eventType,
		Payload: payload,
		ReqID:   reqID,
		TS:      time.Now().UnixMilli(),
	}
	buf, err := json.Marshal(out)
	if err != nil {
		c.logger.Errorw("marshal event",
			logx.Field("type", eventType),
			logx.Field("err", err))
		return
	}
	c.Send(buf)
}

// Close 优雅关闭连接，幂等。
func (c *Client) Close() {
	c.mu.Lock()
	if c.isClosed {
		c.mu.Unlock()
		return
	}
	c.isClosed = true
	close(c.closed)
	c.mu.Unlock()
	_ = c.conn.Close()
}

// Closed 返回连接关闭通知 channel。
func (c *Client) Closed() <-chan struct{} { return c.closed }

// Run 启动读写循环；必须由调用方在 goroutine 中调用。
// 该方法阻塞直到 readPump / writePump 结束。
func (c *Client) Run(ctx context.Context) {
	go c.writePump(ctx)
	c.readPump(ctx)
}

// readPump 阻塞读消息并投递到 Room.inbox。
func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.room.OnClientGone(c)
		c.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure,
				websocket.CloseAbnormalClosure,
			) && !isClosedConnErr(err) {
				c.logger.Errorw("ws read",
					logx.Field("playerId", c.playerID),
					logx.Field("err", err))
			}
			return
		}
		var env protocol.Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			c.logger.Errorw("ws decode envelope",
				logx.Field("playerId", c.playerID),
				logx.Field("err", err))
			continue
		}
		c.room.Submit(Intent{
			Type:    env.Type,
			Payload: env.Payload,
			ReqID:   env.ReqID,
			From:    c,
		})
	}
}

// writePump 消费 send 队列并写入 WebSocket；定时发 ping 检测对端存活。
func (c *Client) writePump(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.closed:
			return
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func isClosedConnErr(err error) bool {
	if err == nil {
		return false
	}
	var ne *net.OpError
	if errors.As(err, &ne) {
		return true
	}
	return false
}
