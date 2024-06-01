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

	"github.com/kurosann/aqt-sdk/ws"
)

type SvcType string

const (
	Public   SvcType = "Public"
	Private  SvcType = "Private"
	Business SvcType = "Business"
)

type ILogger interface {
	Infof(template string, args ...interface{})
	Debugf(template string, args ...interface{})
	Warnf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
	Panicf(template string, args ...interface{})
}

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

type WsClient struct {
	typ         SvcType
	conn        *ws.Conn
	url         BaseURL
	keyConfig   KeyConfig
	log         ILogger
	readMonitor func(arg Arg)
	locker      sync.RWMutex
	loginLocker sync.RWMutex
	isLogin     bool
	proxy       func(req *http.Request) (*url.URL, error)
}

func NewBaseWsClient(typ SvcType, url BaseURL, keyConfig KeyConfig, proxy func(req *http.Request) (*url.URL, error)) *WsClient {
	return &WsClient{typ: typ, url: url, keyConfig: keyConfig, proxy: proxy, log: DefaultLogger{}}
}

type ExchangeClient struct {
	*PublicClient
	*BusinessClient
	*PrivateClient
}

func (w *WsClient) send(data any) error {
	bs, err := json.Marshal(data)
	if err != nil {
		return err
	}
	w.log.Infof(fmt.Sprintf("%s:send %s", w.typ, string(bs)))
	return w.conn.Write(bs)
}

func (w *WsClient) login(ctx context.Context) error {
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
		Args: []map[string]string{w.keyConfig.makeWsSign()},
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
func (w *WsClient) subscribe(ctx context.Context, arg *Arg, callback func(resp *WsOriginResp)) error {
	defer w.send(Op{Op: "unsubscribe", Args: []*Arg{arg}})
	return w.watch(ctx, arg.Key(), Op{Op: "subscribe", Args: []*Arg{arg}}, callback)
}

// 取消订阅
func (w *WsClient) unsubscribe(arg *Arg) error {
	return w.send(Op{Op: "unsubscribe", Args: []*Arg{arg}})
}

// 监听
func (w *WsClient) watch(ctx context.Context, key string, op Op, callback func(resp *WsOriginResp)) error {
	if err := w.CheckConn(); err != nil {
		return err
	}
	if err := w.send(op); err != nil {
		return err
	}

	// 注册监听
	ch := w.conn.RegisterWatch(key)
	defer func() {
		// 返回则取消监听
		w.conn.UnregisterWatch(key)
	}()

	for {
		// 并发控制
		select {
		case <-ctx.Done():
			return nil
		case <-w.conn.Context().Done():
			return context.Cause(w.conn.Context())
		case data, ok := <-ch:
			if !ok {
				return errors.New("subscribe by cancel")
			}
			if data.Typ != websocket.TextMessage || string(data.Data) == "pong" {
				continue
			}
			rp := &WsOriginResp{}
			err := json.Unmarshal(data.Data, rp)
			if err != nil {
				w.log.Errorf(fmt.Sprintf("msg:%v, data:%s", err.Error(), string(data.Data)))
				continue
			}
			if rp.Event == "error" {
				w.log.Errorf(fmt.Sprintf("error msg:%v, data:%s", rp.Msg, string(data.Data)))
			}
			if key == rp.Arg.Key() || rp.Event == key {
				callback(rp)
			}
		}
	}
}

// 判断连接存活的条件
func (w *WsClient) isAlive() bool {
	return w.conn != nil &&
		w.conn.Status == ws.Alive
}

// CheckConn 检查连接是否健康
func (w *WsClient) CheckConn() error {
	if !w.isAlive() {
		w.locker.Lock()
		defer w.locker.Unlock()
		if w.isAlive() {
			return nil
		}

		conn, err := ws.DialContext(context.Background(), string(w.url), ws.WithProxy(w.proxy))
		if err != nil {
			return err
		}
		w.conn = conn
		w.isLogin = false
	}
	return nil
}

func NewWsClient(ctx context.Context, keyConfig KeyConfig, env Destination, proxy ...string) *ExchangeClient {
	return NewWsClientWithCustom(ctx, keyConfig, env, DefaultWsUrls, proxy...)
}

func NewWsClientWithCustom(ctx context.Context, keyConfig KeyConfig, env Destination, urls map[Destination]map[SvcType]BaseURL, proxy ...string) *ExchangeClient {
	proxyURL := http.ProxyFromEnvironment
	if len(proxy) != 0 && proxy[0] != "" {
		parse, err := url.Parse(proxy[0])
		if err != nil {
			panic(err.Error())
		}
		proxyURL = http.ProxyURL(parse)
	}
	pc := &PublicClient{*NewBaseWsClient(Public, urls[env][Public], keyConfig, proxyURL)}
	bc := &BusinessClient{*NewBaseWsClient(Business, urls[env][Business], keyConfig, proxyURL)}
	pvc := &PrivateClient{*NewBaseWsClient(Private, urls[env][Private], keyConfig, proxyURL)}
	return &ExchangeClient{
		pc,
		bc,
		pvc,
	}
}

func (w *ExchangeClient) SetReadMonitor(readMonitor func(arg Arg)) {
	w.PublicClient.readMonitor = readMonitor
	w.BusinessClient.readMonitor = readMonitor
	w.PrivateClient.readMonitor = readMonitor
}
func (w *ExchangeClient) SetLog(log ILogger) {
	w.PublicClient.log = log
	w.BusinessClient.log = log
	w.PrivateClient.log = log
}
func (w *ExchangeClient) Close() {

}

func Subscribe[T any](c *WsClient, ctx context.Context, arg *Arg, callback func(resp *WsResp[T])) error {
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
