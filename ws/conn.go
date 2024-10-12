package ws

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Status string

const (
	Alive = "alive"
	Dead  = "dead"
)

type MsgType int

const (
	TextMessage   MsgType = 1
	BinaryMessage MsgType = 2
	CloseMessage  MsgType = 8
	PingMessage   MsgType = 9
	PongMessage   MsgType = 10
)

// Conn ws连接
type Conn struct {
	ctx              context.Context
	cancel           context.CancelCauseFunc
	Status           Status
	conn             *websocket.Conn
	dataCh           map[string]chan Data
	lock             sync.RWMutex
	proxy            func(req *http.Request) (*url.URL, error)
	writeTimeout     time.Duration // 写入超时时间
	keepaliveTimeout time.Duration // 响应超时时间
	mt               MsgType       // 连接使用的消息类型
	keepalivePingFn  func(*Conn) error
	keepalivePongFn  func([]byte) bool
}

// DialContext 拨号 使用context控制
func DialContext(ctx context.Context, address string, opts ...Option) (*Conn, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	c := &Conn{
		ctx:              ctx,
		cancel:           cancel,
		Status:           Alive,
		dataCh:           map[string]chan Data{},
		writeTimeout:     time.Second * 3,
		keepaliveTimeout: time.Second * 30,
		mt:               TextMessage,
		keepalivePingFn:  func(conn *Conn) error { return nil },
		keepalivePongFn:  func(bytes []byte) bool { return true },
	}
	for _, opt := range opts {
		opt(c)
	}
	dialer := websocket.DefaultDialer
	if c.proxy != nil {
		dialer.Proxy = c.proxy
	}
	dialer.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	header := http.Header{}
	header.Add("User-Agent", "Go websockets/13.0")
	conn, rp, err := dialer.DialContext(ctx, address, header)
	if err != nil {
		return nil, err
	}
	if rp.StatusCode != 101 {
		return nil, fmt.Errorf("connect response err: %v", rp)
	}
	c.conn = conn
	go c.keepalive()
	go c.read()
	return c, nil
}

func (c *Conn) Context() context.Context {
	return c.ctx
}
func (c *Conn) Close(err error) {
	c.cancel(err)
	c.Status = Dead
	_ = c.conn.Close()
}

// keepalive
func (c *Conn) keepalive() {
	c.conn.SetPongHandler(func(appData string) error {
		c.Status = Alive
		return nil
	})
	for {
		if c.Status == Dead {
			return
		}
		err := c.conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(time.Second*3))
		if err != nil {
			c.Close(err)
			return
		}
		err = c.keepalivePingFn(c)
		if err != nil {
			c.Close(err)
			return
		}
		time.Sleep(c.keepaliveTimeout - time.Second*10)
	}
}

type Data struct {
	Typ  MsgType
	Data []byte
}

// RegisterWatch 注册监听
func (c *Conn) RegisterWatch(id string) <-chan Data {
	c.lock.Lock()
	defer c.lock.Unlock()

	data := make(chan Data, 1)
	c.dataCh[id] = data
	return data
}

// UnregisterWatch 注销监听
func (c *Conn) UnregisterWatch(id string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// 关闭、通知并回收资源
	if ch, ok := c.dataCh[id]; ok {
		close(ch)
	}
	delete(c.dataCh, id)
}

// Write 写入数据
func (c *Conn) Write(data []byte) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	err := c.conn.WriteMessage(int(c.mt), data)
	if err != nil {
		if context.Cause(c.ctx) != nil {
			err = context.Cause(c.ctx)
		}
		c.Close(err)
		return err
	}
	return nil
}

// 读取数据流
func (c *Conn) read() {
	for {
		// 并发控制
		select {
		case <-c.ctx.Done():
			return
		default:
		}
		// 设置读取超时 超过读取不到任何数据将关闭连接
		_ = c.conn.SetReadDeadline(time.Now().Add(c.keepaliveTimeout))
		mt, data, err := c.conn.ReadMessage()
		if err != nil {
			c.Close(err)
			return
		}
		if c.keepalivePongFn(data) {
			c.Status = Alive
			continue
		}
		// 写入已注册的监听中
		c.lock.RLock()
		for _, ch := range c.dataCh {
			ch <- Data{
				Typ:  MsgType(mt),
				Data: data,
			}
		}
		c.lock.RUnlock()
	}
}
