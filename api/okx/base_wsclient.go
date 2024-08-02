package okx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/kurosann/aqt-sdk/api"
	"github.com/kurosann/aqt-sdk/ws"
)

type DefaultLogger struct {
}

func (w DefaultLogger) Infof(template string, args ...interface{}) {
	log.Printf(template, args...)
}

func (w DefaultLogger) Debugf(template string, args ...interface{}) {
	log.Printf(template, args...)
}

func (w DefaultLogger) Warnf(template string, args ...interface{}) {
	log.Printf(template, args...)
}

func (w DefaultLogger) Errorf(template string, args ...interface{}) {
	log.Printf(template, args...)
}

func (w DefaultLogger) Panicf(template string, args ...interface{}) {
	log.Printf(template, args...)
}

type BaseWsClient struct {
	ctx         context.Context
	typ         SvcType
	conn        *ws.Conn
	url         BaseURL
	keyConfig   OkxKeyConfig
	Log         api.ILogger
	ReadMonitor func(arg Arg)
	locker      sync.RWMutex
	loginLocker sync.RWMutex
	isLogin     bool
	proxy       func(req *http.Request) (*url.URL, error)
	callbacks   map[string]func(resp *WsOriginResp)
}

func NewBaseWsClient(ctx context.Context, typ SvcType, url BaseURL, keyConfig OkxKeyConfig, proxy func(req *http.Request) (*url.URL, error)) BaseWsClient {
	return BaseWsClient{
		ctx:         ctx,
		typ:         typ,
		url:         url,
		keyConfig:   keyConfig,
		proxy:       proxy,
		Log:         DefaultLogger{},
		callbacks:   map[string]func(resp *WsOriginResp){},
		ReadMonitor: func(arg Arg) {},
	}
}

func (w *BaseWsClient) send(data any) error {
	bs, err := json.Marshal(data)
	if err != nil {
		return err
	}
	w.Log.Infof(fmt.Sprintf("%s:send %s", w.typ, string(bs)))
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
			w.Log.Errorf(fmt.Sprintf("%s:read ws err: %v", w.typ, rp))
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
	err = w.watch(ctx, arg.Key(), Op{Op: "subscribe", Args: []*Arg{arg}}, callback)
	if err != nil {
		return err
	}
	defer w.send(Op{Op: "unsubscribe", Args: []*Arg{arg}})
	return nil
}

// 取消订阅
func (w *BaseWsClient) Unsubscribe(arg *Arg) error {
	return w.send(Op{Op: "unsubscribe", Args: []*Arg{arg}})
}
func (w *BaseWsClient) receive() {
	ch := w.conn.RegisterWatch("receive")
	defer w.conn.UnregisterWatch("receive")
	for {
		select {
		case <-w.conn.Context().Done():
			return
		case data, ok := <-ch:
			if !ok {
				return
			}
			if data.Typ != websocket.TextMessage {
				continue
			}
			rp := &WsOriginResp{}
			err := json.Unmarshal(data.Data, rp)
			if err != nil {
				w.Log.Errorf(fmt.Sprintf("msg:%v, data:%s", err.Error(), string(data.Data)))
				continue
			}
			if rp.Event == "error" {
				w.Log.Errorf(fmt.Sprintf("error msg:%v, data:%s", rp.Msg, string(data.Data)))
				w.conn.Close(errors.New(rp.Msg))
			}
			w.ReadMonitor(rp.Arg)
			if callback, ok := w.getWatch(rp.Arg.Key()); ok {
				callback(rp)
			}
			if callback, ok := w.getWatch(rp.Event); ok {
				callback(rp)
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
		w.conn.Status == ws.Alive
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
		conn, err := ws.DialContext(w.ctx, string(w.url), ws.WithProxy(w.proxy))
		if err != nil {
			return err
		}
		// 保持连接的依据
		conn.SetKeepAlive(
			func(conn *ws.Conn) error {
				return conn.Write([]byte("ping"))
			},
			func(data []byte) bool {
				return string(data) == "pong"
			})
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
			c.Log.Errorf(err.Error())
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
