package okx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kurosann/aqt-sdk/api"
	"github.com/kurosann/aqt-sdk/ws"
)

const (
	reconnectDelay    = 5 * time.Second
	keepAliveInterval = 30 * time.Second
)

type WsInfo struct {
	typ         SvcType
	url         BaseURL
	keyConfig   OkxKeyConfig
	proxy       func(req *http.Request) (*url.URL, error)
	readMonitor func(arg Arg)
	log         api.ILogger
}

type BaseWsClient struct {
	ctx           context.Context
	cancel        context.CancelFunc
	typ           SvcType
	conn          *ws.Conn
	url           BaseURL
	keyConfig     OkxKeyConfig
	log           api.ILogger
	readMonitor   func(arg Arg)
	locker        sync.RWMutex
	loginLocker   sync.RWMutex
	isLogin       bool
	proxy         func(req *http.Request) (*url.URL, error)
	callbacks     map[string]func(resp *WsOriginResp)
	reconnecting  atomic.Bool
	closeOnce     sync.Once
	subscriptions sync.Map // 存储当前活跃的订阅
}

func NewBaseWsClient(ctx context.Context, info WsInfo) *BaseWsClient {
	ctx, cancel := context.WithCancel(ctx)
	client := &BaseWsClient{
		ctx:         ctx,
		cancel:      cancel,
		typ:         info.typ,
		url:         info.url,
		keyConfig:   info.keyConfig,
		proxy:       info.proxy,
		log:         info.log,
		callbacks:   make(map[string]func(resp *WsOriginResp)),
		readMonitor: info.readMonitor,
	}

	// 启动自动重连
	go client.autoReconnect()

	return client
}

func (w *BaseWsClient) send(data any) error {
	bs, err := json.Marshal(data)
	if err != nil {
		return err
	}
	w.log.Infof(fmt.Sprintf("%s:send %s", w.typ, string(bs)))
	return w.conn.Write(bs)
}

func (w *BaseWsClient) Login(ctx context.Context) error {
	w.loginLocker.Lock()
	defer w.loginLocker.Unlock()
	if w.isLogin {
		return nil
	}

	// 并发控制
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	// 登录并监听
	err := w.watch(ctx, "login", Op{
		Op:   "login",
		Args: []map[string]string{w.keyConfig.MakeWsSign()},
	}, func(rp *WsOriginResp) {
		if rp.Code != "0" {
			w.log.Errorf(fmt.Sprintf("%s:read ws err: %v", w.typ, rp))
			cancel(errors.New(rp.Msg))
		} else {
			cancel(nil)
		}
	})
	if err != nil {
		return err
	}
	// 等待登录完成
	<-ctx.Done()
	// 处理登录错误
	err = context.Cause(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	w.isLogin = true
	return nil
}

// 订阅
func (w *BaseWsClient) subscribe(ctx context.Context, arg *Arg, callback func(resp *WsOriginResp)) (err error) {
	// 保存订阅信息
	w.subscriptions.Store(arg.Key(), subscriptionInfo{
		arg:      arg,
		callback: callback,
	})
	defer w.subscriptions.Delete(arg.Key())

	err = w.watch(ctx, arg.Key(), Op{Op: "subscribe", Args: []*Arg{arg}}, callback)
	if err != nil {
		return err
	}
	defer w.send(Op{Op: "unsubscribe", Args: []*Arg{arg}})
	return nil
}

// 取消订阅
func (w *BaseWsClient) Unsubscribe(arg *Arg) error {
	w.subscriptions.Delete(arg.Key())
	return w.send(Op{Op: "unsubscribe", Args: []*Arg{arg}})
}

func (w *BaseWsClient) receive() {
	ch, err := w.conn.RegisterWatch("receive")
	if err != nil {
		w.log.Errorf("%s: failed to register watch: %v", w.typ, err)
		return
	}
	defer w.conn.UnregisterWatch("receive")

	for {
		select {
		case <-w.ctx.Done():
			return
		case data, ok := <-ch:
			if !ok {
				return
			}

			if err := w.handleMessage(data); err != nil {
				w.log.Errorf("%s: message handling error: %v", w.typ, err)
			}
		}
	}
}

// 监听
func (w *BaseWsClient) watch(ctx context.Context, key string, op Op, callback func(resp *WsOriginResp)) error {
	if err := w.CheckConn(); err != nil {
		return err
	}
	if err := w.send(op); err != nil {
		return err
	}

	// 注册监听
	w.registerWatch(key, callback)
	// 返回则取消监听
	defer w.unregisterWatch(key)
	for {
		// 并发控制
		select {
		case <-ctx.Done():
			return nil
		case <-w.conn.Context().Done():
			return context.Cause(w.conn.Context())
		}
	}
}

// 判断连接存活的条件
func (w *BaseWsClient) isAlive() bool {
	return w.conn != nil &&
		w.conn.Status() == ws.Alive
}

// CheckConn 检查连接是否健康
func (w *BaseWsClient) CheckConn() error {
	if !w.isAlive() {
		w.locker.Lock()
		defer w.locker.Unlock()
		if w.isAlive() {
			return nil
		}

		// 非健康情况重新进行拨号
		conn, err := ws.DialContext(w.ctx,
			string(w.url),
			ws.WithProxy(w.proxy),
			ws.WithKeepalive(time.Second*30,
				func(conn *ws.Conn) error { return conn.Write([]byte("ping")) },
				func(data []byte) bool { return string(data) == "pong" }))
		if err != nil {
			return err
		}
		w.conn = conn
		w.isLogin = false
		go w.receive()
	}
	return nil
}

func (w *BaseWsClient) registerWatch(key string, callback func(resp *WsOriginResp)) {
	w.locker.Lock()
	defer w.locker.Unlock()

	w.callbacks[key] = callback
}

func (w *BaseWsClient) unregisterWatch(key string) {
	w.locker.Lock()
	defer w.locker.Unlock()

	delete(w.callbacks, key)
}

func (w *BaseWsClient) getWatch(key string) (func(resp *WsOriginResp), bool) {
	w.locker.RLock()
	defer w.locker.RUnlock()

	f, ok := w.callbacks[key]
	return f, ok
}

func Subscribe[T any](c *BaseWsClient, ctx context.Context, arg *Arg, callback func(resp *WsResp[T])) error {
	return c.subscribe(ctx, arg, func(resp *WsOriginResp) {
		if resp.Event == "subscribe" {
			return
		}
		var t []T
		err := json.Unmarshal(resp.Data, &t)
		if err != nil {
			c.log.Errorf(err.Error())
			return
		}
		callback(&WsResp[T]{
			Event:  resp.Event,
			ConnId: resp.ConnId,
			Code:   resp.Code,
			Msg:    resp.Msg,
			Arg:    resp.Arg,
			Data:   t,
		})
	})
}

func (w *BaseWsClient) Close() {
	w.closeOnce.Do(func() {
		w.cancel()
		if w.conn != nil {
			w.conn.Close(nil)
		}
	})
}

func (w *BaseWsClient) autoReconnect() {
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-time.After(reconnectDelay):
			if !w.isAlive() && !w.reconnecting.Load() {
				w.reconnecting.Store(true)
				w.log.Infof("%s: attempting to reconnect...", w.typ)

				if err := w.CheckConn(); err != nil {
					w.log.Errorf("%s: reconnection failed: %v", w.typ, err)
					continue
				}

				// 重新登录
				if err := w.Login(w.ctx); err != nil {
					w.log.Errorf("%s: relogin failed: %v", w.typ, err)
					continue
				}

				// 重新订阅之前的频道
				w.resubscribeChannels()
				w.reconnecting.Store(false)
			}
		}
	}
}

func (w *BaseWsClient) handleMessage(data ws.Data) error {
	if data.Type != ws.TextMessage {
		return nil
	}

	rp := &WsOriginResp{}
	if err := json.Unmarshal(data.Data, rp); err != nil {
		return fmt.Errorf("unmarshal error: %w, data: %s", err, string(data.Data))
	}

	if rp.Event == "error" {
		err := errors.New(rp.Msg)
		w.conn.Close(err)
		return err
	}

	w.readMonitor(rp.Arg)

	// 处理回调
	if callback, ok := w.getWatch(rp.Arg.Key()); ok {
		callback(rp)
	}
	if callback, ok := w.getWatch(rp.Event); ok {
		callback(rp)
	}

	return nil
}

func (w *BaseWsClient) resubscribeChannels() {
	w.subscriptions.Range(func(key, value interface{}) bool {
		info := value.(subscriptionInfo)

		// 为每个订阅创建新的上下文
		ctx, cancel := context.WithTimeout(w.ctx, 10*time.Second)
		defer cancel()

		// 重新订阅
		err := w.watch(ctx, info.arg.Key(), Op{
			Op:   "subscribe",
			Args: []*Arg{info.arg},
		}, info.callback)

		if err != nil {
			w.log.Errorf("%s: failed to resubscribe channel %s: %v", w.typ, info.arg.Key(), err)
			return true // 继续处理下一个订阅
		}

		w.log.Infof("%s: successfully resubscribed to channel %s", w.typ, info.arg.Key())
		return true
	})
}

type subscriptionInfo struct {
	arg      *Arg
	callback func(resp *WsOriginResp)
}
