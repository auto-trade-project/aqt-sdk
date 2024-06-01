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
	ctx          context.Context
	cancel       context.CancelCauseFunc
	Status       Status
	conn         *websocket.Conn
	dataCh       map[string]chan Data
	lock         sync.RWMutex
	proxy        func(req *http.Request) (*url.URL, error)
	writeTimeout time.Duration // 写入超时时间
	mt           MsgType       // 连接使用的消息类型
}

// DialContext 拨号 使用context控制
func DialContext(ctx context.Context, address string, opts ...Option) (*Conn, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	c := &Conn{
		ctx:          ctx,
		cancel:       cancel,
		Status:       Dead,
		dataCh:       map[string]chan Data{},
		writeTimeout: time.Second * 3,
		mt:           TextMessage,
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

// keepalive
func (c *Conn) keepalive() {
	for {
		if c.Status == Dead {
			return
		}
		err := c.conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(c.writeTimeout))
		if err != nil {
			c.Status = Dead
			c.cancel(err)
			_ = c.conn.Close()
			return
		}
		c.conn.SetPongHandler(func(appData string) error {
			c.Status = Alive
			return nil
		})
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
	close(c.dataCh[id])
	delete(c.dataCh, id)
}

// Write 写入数据
func (c *Conn) Write(data []byte) error {
	_ = c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	err := c.conn.WriteMessage(int(c.mt), data)
	if err != nil {
		c.Status = Dead
		c.cancel(err)
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
