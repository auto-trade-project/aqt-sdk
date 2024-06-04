package ws

import (
	"context"
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
	ctx               context.Context
	cancel            context.CancelCauseFunc
	Status            Status
	conn              *websocket.Conn
	dataCh            map[string]chan Data
	lock              sync.RWMutex
	proxy             func(req *http.Request) (*url.URL, error)
	writeTimeout      time.Duration // 写入超时时间
	mt                MsgType       // 连接使用的消息类型
	keepaliveFn       func(*Conn) error
	keepaliveListenFn func([]byte) bool
}

// DialContext 拨号 使用context控制
func DialContext(ctx context.Context, address string, opts ...Option) (*Conn, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	c := &Conn{
		ctx:               ctx,
		cancel:            cancel,
		Status:            Alive,
		dataCh:            map[string]chan Data{},
		writeTimeout:      time.Second * 3,
		mt:                TextMessage,
		keepaliveFn:       func(conn *Conn) error { return nil },
		keepaliveListenFn: func(bytes []byte) bool { return true },
	}
	for _, opt := range opts {
		opt(c)
	}
	dialer := websocket.DefaultDialer
	if c.proxy != nil {
		dialer.Proxy = c.proxy
	}
	conn, rp, err := dialer.DialContext(ctx, address, nil)
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
func (c *Conn) SetKeepAlive(keepaliveFn func(conn *Conn) error, keepaliveListenFn func(data []byte) bool) {
	c.keepaliveFn = keepaliveFn
	c.keepaliveListenFn = keepaliveListenFn
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
			c.Status = Dead
			c.cancel(err)
			_ = c.conn.Close()
			return
		}
		err = c.keepaliveFn(c)
		if err != nil {
			c.Status = Dead
			c.cancel(err)
			_ = c.conn.Close()
			return
		}
		time.Sleep(time.Second * 20)
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
		c.Status = Dead
		if context.Cause(c.ctx) != nil {
			err = context.Cause(c.ctx)
		} else {
			c.cancel(err)
		}
		_ = c.conn.Close()
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
		mt, data, err := c.conn.ReadMessage()
		if err != nil {
			c.Status = Dead
			c.cancel(err)
			_ = c.conn.Close()
			return
		}
		if c.keepaliveListenFn(data) {
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
