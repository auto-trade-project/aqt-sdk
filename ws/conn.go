package ws

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	ErrConnClosed    = errors.New("connection closed")
	ErrWriteTimeout  = errors.New("write timeout")
	ErrReadTimeout   = errors.New("read timeout")
	ErrInvalidStatus = errors.New("invalid connection status")
)

type Status string

const (
	Alive Status = "alive"
	Dead  Status = "dead"
)

type MsgType int

const (
	TextMessage   MsgType = 1
	BinaryMessage MsgType = 2
	CloseMessage  MsgType = 8
	PingMessage   MsgType = 9
	PongMessage   MsgType = 10

	defaultWriteTimeout     = 3 * time.Second
	defaultKeepaliveTimeout = 30 * time.Second
	defaultPingInterval     = 20 * time.Second // keepaliveTimeout - 10s
)

// Conn ws连接
type Conn struct {
	ctx    context.Context
	cancel context.CancelCauseFunc

	mu     sync.RWMutex
	status Status
	conn   *websocket.Conn
	dataCh map[string]chan Data

	config connConfig
}

type connConfig struct {
	proxy            func(req *http.Request) (*url.URL, error)
	writeTimeout     time.Duration
	keepaliveTimeout time.Duration
	msgType          MsgType
	keepalivePingFn  func(*Conn) error
	keepalivePongFn  func([]byte) bool
}

type Data struct {
	Type MsgType
	Data []byte
}

// NewConn 创建新的websocket连接
func DialContext(ctx context.Context, address string, opts ...Option) (*Conn, error) {
	ctx, cancel := context.WithCancelCause(ctx)

	c := &Conn{
		ctx:    ctx,
		cancel: cancel,
		status: Alive,
		dataCh: make(map[string]chan Data),
		config: connConfig{
			writeTimeout:     defaultWriteTimeout,
			keepaliveTimeout: defaultKeepaliveTimeout,
			msgType:          TextMessage,
			keepalivePingFn:  func(conn *Conn) error { return nil },
			keepalivePongFn:  func(bytes []byte) bool { return true },
		},
	}

	// 应用选项
	for _, opt := range opts {
		opt(c)
	}

	if err := c.dial(address); err != nil {
		return nil, fmt.Errorf("dial websocket: %w", err)
	}

	// 启动后台任务
	go c.keepalive()
	go c.readPump()

	return c, nil
}

func (c *Conn) dial(address string) error {
	dialer := websocket.DefaultDialer
	if c.config.proxy != nil {
		dialer.Proxy = c.config.proxy
	}
	dialer.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	header := http.Header{}
	header.Add("User-Agent", "Go websockets/13.0")

	conn, resp, err := dialer.DialContext(c.ctx, address, header)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	c.conn = conn
	return nil
}

func (c *Conn) Context() context.Context {
	return c.ctx
}

func (c *Conn) Status() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

func (c *Conn) Close(err error) {
	c.mu.Lock()
	if c.status == Dead {
		c.mu.Unlock()
		return
	}
	c.status = Dead
	c.mu.Unlock()

	c.cancel(err)
	_ = c.conn.Close()

	// 关闭所有数据通道
	c.mu.Lock()
	for _, ch := range c.dataCh {
		close(ch)
	}
	c.dataCh = nil
	c.mu.Unlock()
}

// keepalive 心跳保活
func (c *Conn) keepalive() {
	ticker := time.NewTicker(defaultPingInterval)
	defer ticker.Stop()

	c.conn.SetPongHandler(func(string) error {
		c.mu.Lock()
		c.status = Alive
		c.mu.Unlock()
		return nil
	})

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if c.Status() == Dead {
				return
			}
			if err := c.ping(); err != nil {
				c.Close(fmt.Errorf("ping failed: %w", err))
				return
			}
		}
	}
}

func (c *Conn) ping() error {
	deadline := time.Now().Add(c.config.writeTimeout)
	err := c.conn.WriteControl(websocket.PingMessage, []byte("ping"), deadline)
	if err != nil {
		return err
	}
	return c.config.keepalivePingFn(c)
}

// RegisterWatch 注册消息监听
func (c *Conn) RegisterWatch(id string) (<-chan Data, error) {
	if c.Status() == Dead {
		return nil, ErrConnClosed
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan Data, 1)
	c.dataCh[id] = ch
	return ch, nil
}

// UnregisterWatch 注销消息监听
func (c *Conn) UnregisterWatch(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ch, exists := c.dataCh[id]; exists {
		close(ch)
		delete(c.dataCh, id)
	}
}

// Write 发送消息
func (c *Conn) Write(data []byte) error {
	if c.Status() == Dead {
		return ErrConnClosed
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	deadline := time.Now().Add(c.config.writeTimeout)
	if err := c.conn.SetWriteDeadline(deadline); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}

	if err := c.conn.WriteMessage(int(c.config.msgType), data); err != nil {
		if ctxErr := context.Cause(c.ctx); ctxErr != nil {
			err = ctxErr
		}
		c.Close(err)
		return err
	}
	return nil
}

// readPump 读取消息泵
func (c *Conn) readPump() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if err := c.readMessage(); err != nil {
				c.Close(err)
				return
			}
		}
	}
}

func (c *Conn) readMessage() error {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.config.keepaliveTimeout)); err != nil {
		return fmt.Errorf("set read deadline: %w", err)
	}

	msgType, data, err := c.conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read message: %w", err)
	}

	if c.config.keepalivePongFn(data) {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.status = Alive
		return nil
	}

	c.mu.RLock()
	for _, ch := range c.dataCh {
		ch <- Data{Type: MsgType(msgType), Data: data}
	}
	c.mu.RUnlock()

	return nil
}
