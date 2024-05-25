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
	"time"

	"github.com/gorilla/websocket"
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

type WsClientConn struct {
	parentCtx           context.Context
	ctx                 context.Context
	cancel              context.CancelCauseFunc
	typ                 SvcType
	url                 BaseURL
	conn                *websocket.Conn
	SubscribeCallback   func()
	UnSubscribeCallback func()
	callbackMap         map[string]func(*WsOriginResp)
	isLogin             bool
	log                 ILogger
	proxy               func(*http.Request) (*url.URL, error)
	isClose             bool
	chanLock            sync.RWMutex
	subscribeLock       sync.RWMutex
	loginLock           sync.RWMutex
	readMonitor         func(arg Arg)
	closeListen         func()
	subArgs             map[string]*Arg
	keyConfig           KeyConfig
}
type WsClient struct {
	Conns               map[SvcType]*WsClientConn
	ctx                 context.Context
	cancel              func()
	SubscribeCallback   func()
	UnSubscribeCallback func()
	keyConfig           KeyConfig
	log                 ILogger
	closeListen         func()
	readMonitor         func(arg Arg)
	proxy               func(*http.Request) (*url.URL, error)
}

func NewWsClient(ctx context.Context, keyConfig KeyConfig, env Destination, proxy ...string) *WsClient {
	return NewWsClientWithCustom(ctx, keyConfig, env, DefaultWsUrls, proxy...)
}

func NewWsClientWithCustom(ctx context.Context, keyConfig KeyConfig, env Destination, urls map[Destination]map[SvcType]BaseURL, proxy ...string) *WsClient {
	ctx, cancel := context.WithCancel(ctx)
	proxyURL := http.ProxyFromEnvironment
	if len(proxy) != 0 && proxy[0] != "" {
		parse, err := url.Parse(proxy[0])
		if err != nil {
			panic(err.Error())
		}
		proxyURL = http.ProxyURL(parse)
	}
	conns := map[SvcType]*WsClientConn{}
	for svcType, baseURL := range urls[env] {
		ctxChild, cancelChild := context.WithCancelCause(ctx)
		conns[svcType] = &WsClientConn{
			parentCtx:           ctx,
			ctx:                 ctxChild,
			cancel:              cancelChild,
			typ:                 svcType,
			url:                 baseURL,
			SubscribeCallback:   func() {},
			UnSubscribeCallback: func() {},
			callbackMap:         make(map[string]func(*WsOriginResp)),
			log:                 DefaultLogger{},
			proxy:               proxyURL,
			readMonitor:         func(Arg) {},
			closeListen:         func() {},
			keyConfig:           keyConfig,
			subArgs:             map[string]*Arg{},
		}
	}
	return &WsClient{
		ctx:         ctx,
		cancel:      cancel,
		keyConfig:   keyConfig,
		Conns:       conns,
		log:         DefaultLogger{},
		closeListen: func() {},
		readMonitor: func(arg Arg) {},
		proxy:       proxyURL,
	}
}
func (w *WsClientConn) connect(ctx context.Context, url BaseURL) (*websocket.Conn, *http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	dialer := websocket.DefaultDialer
	dialer.Proxy = w.proxy
	conn, rp, err := dialer.DialContext(ctx, string(url), nil)
	if err != nil {
		return nil, nil, err
	}
	go w.heartbeat()
	return conn, rp, err
}

func (w *WsClientConn) heartbeat() {
	timer := time.NewTicker(time.Second * 20)
	defer timer.Stop()
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-timer.C:
			w.conn.SetWriteDeadline(time.Now().Add(time.Second))
			err := w.conn.WriteMessage(websocket.TextMessage, []byte("ping"))
			if err != nil {
				w.log.Errorf(err.Error())
				w.Close(err)
				return
			}
		}
	}
}

func (w *WsClient) SetReadMonitor(readMonitor func(arg Arg)) {
	w.readMonitor = readMonitor
	for _, conn := range w.Conns {
		conn.readMonitor = readMonitor
	}
}
func (w *WsClient) SetCloseListen(closeListen func()) {
	w.closeListen = closeListen
	for _, conn := range w.Conns {
		conn.closeListen = closeListen
	}
}
func (w *WsClient) SetLog(log ILogger) {
	w.log = log
	for _, conn := range w.Conns {
		conn.log = log
	}
}
func (w *WsClient) Close() {

	for k, conn := range w.Conns {
		conn.Close(nil)
		delete(w.Conns, k)
	}
	w.closeListen()
}
func (w *WsClientConn) Close(err error) {
	if w.isClose {
		return
	}
	if w.conn != nil {
		w.conn.Close()
	}

	w.isClose = true
	w.closeListen()
	w.cancel(err)
}
func (w *WsClientConn) lazyConnect() (*websocket.Conn, error) {
	if w.conn == nil || w.isClose {
		w.ctx, w.cancel = context.WithCancelCause(context.Background())
		c, rp, err := w.connect(w.ctx, w.url)
		if err != nil {
			return nil, err
		}
		if rp.StatusCode != 101 {
			return nil, fmt.Errorf("connect response err: %v", rp)
		}
		// 监听连接关闭,进行资源回收
		c.SetCloseHandler(func(code int, text string) error {
			w.Close(errors.New("ws conn is close"))
			return nil
		})
		w.isClose, w.isLogin = false, false
		w.conn = c

		go w.process()
	}
	return w.conn, nil
}
func (w *WsClientConn) push(typ SvcType, arg Arg, resp *WsOriginResp) {
	if callback, ok := w.GetCallback(arg.Key()); ok {
		callback(resp)
	} else {
		w.log.Warnf(fmt.Sprintf("%s:not found channel, ignore data: %+v", typ, resp))
	}
}

func (w *WsClientConn) RegCallback(channel string, c func(*WsOriginResp)) {
	w.chanLock.Lock()
	defer w.chanLock.Unlock()

	w.callbackMap[channel] = c
}
func (w *WsClientConn) GetCallback(channel string) (func(*WsOriginResp), bool) {
	w.chanLock.RLock()
	defer w.chanLock.RUnlock()

	if respCh, ok := w.callbackMap[channel]; ok {
		return respCh, ok
	}
	return nil, false
}

func (w *WsClientConn) UnRegCh(channel string) {
	w.chanLock.Lock()
	defer w.chanLock.Unlock()
	if _, ok := w.callbackMap[channel]; ok {
		delete(w.callbackMap, channel)
	}
}

func (w *WsClientConn) process() {
	defer func() {
		if err := recover(); err != nil {
			w.log.Errorf(fmt.Sprintf("%v", err))
			w.Close(fmt.Errorf("%v", err))
		}
	}()
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}
		mtp, bs, err := w.conn.ReadMessage()
		if err != nil {
			w.log.Errorf(err.Error())
			w.Close(err)
			return
		}
		if mtp != websocket.TextMessage || string(bs) == "pong" {
			continue
		}
		v := &WsOriginResp{}
		err = json.Unmarshal(bs, v)
		if err != nil {
			w.log.Errorf(fmt.Sprintf("msg:%v, data:%s", err.Error(), string(bs)))
			continue
		}
		w.readMonitor(v.Arg)
		if v.Event == "unsubscribe" {
			if w.UnSubscribeCallback != nil {
				w.UnSubscribeCallback()
			}
			continue
		} else if v.Event == "subscribe" {
			if w.SubscribeCallback != nil {
				w.SubscribeCallback()
			}
			w.log.Infof(fmt.Sprintf("%s:%+v subscribe success", w.typ, v.Arg))
		} else if v.Event == "login" {
			w.log.Infof(fmt.Sprintf("%s:login success", w.typ))
			w.isLogin = true
			v.Arg.Channel = "login"
		} else if v.Event == "error" {
			if v.Code == "60009" {
				v.Arg.Channel = "login"
			}
			w.log.Errorf(fmt.Sprintf("%s:read ws err: %v", w.typ, v))
		}
		w.push(w.typ, v.Arg, v)
	}
}

func (w *WsClient) Send(typ SvcType, req any) error {
	conn, err := w.Conns[typ].lazyConnect()
	if err != nil {
		return err
	}
	bs, err := json.Marshal(req)
	if err != nil {
		return err
	}
	w.log.Infof(fmt.Sprintf("%s:send %s", typ, string(bs)))
	if w.Conns[typ].isClose {
		return errors.New("conn is close")
	}
	conn.SetWriteDeadline(time.Now().Add(time.Second * 3))
	return conn.WriteMessage(websocket.TextMessage, bs)
}
func (w *WsClientConn) Send(req any) error {
	conn, err := w.lazyConnect()
	if err != nil {
		return err
	}
	bs, err := json.Marshal(req)
	if err != nil {
		return err
	}
	w.log.Infof(fmt.Sprintf("%s:send %s", w.typ, string(bs)))
	if w.isClose {
		return errors.New("conn is close")
	}
	conn.SetWriteDeadline(time.Now().Add(time.Second * 3))
	return conn.WriteMessage(websocket.TextMessage, bs)
}

func (w *WsClientConn) login() error {
	w.loginLock.Lock()
	defer w.loginLock.Unlock()

	if w.isLogin {
		return nil
	}
	c := make(chan *WsOriginResp, 1)
	w.RegCallback(Arg{
		Channel: "login",
	}.Key(), func(resp *WsOriginResp) {
		c <- resp
	})
	err := w.Send(Op{
		Op:   "login",
		Args: []map[string]string{w.keyConfig.makeWsSign()},
	})
	if err != nil {
		return err
	}
	resp := <-c
	if resp.Event == "error" || !w.isLogin {
		return errors.New(resp.Msg)
	}
	return nil
}
func Subscribe[T any](w *WsClient, arg *Arg, typ SvcType, callback func(resp *WsResp[T]), needLogin bool) error {
	return subscribe(w.Conns[typ], arg, func(resp *WsOriginResp) {
		var t []T
		err := json.Unmarshal(resp.Data, &t)
		if err != nil {
			w.log.Errorf(err.Error())
		}
		callback(&WsResp[T]{
			Event:  resp.Event,
			ConnId: resp.ConnId,
			Code:   resp.Code,
			Msg:    resp.Msg,
			Arg:    resp.Arg,
			Data:   t,
		})
	}, needLogin)
}
func subscribe(w *WsClientConn, arg *Arg, callback func(resp *WsOriginResp), needLogin bool) error {
	w.subscribeLock.Lock()
	defer w.subscribeLock.Unlock()
	w.lazyConnect()
	ctx, cancel := context.WithCancel(w.ctx)
	w.RegCallback(arg.Key(), func(resp *WsOriginResp) {
		if resp.Event == "subscribe" {
			cancel()
			return
		}
		callback(resp)
	})
	w.subArgs[arg.Key()] = arg
	var err error
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				if needLogin && !w.isLogin {
					if err = w.login(); err != nil {
						cancel()
						return
					}
				}
				if err = w.Send(Op{
					Op:   "subscribe",
					Args: []*Arg{arg},
				}); err != nil {
					cancel()
					return
				}
			}
		}
	}()
	<-ctx.Done()
	if err != nil {
		return err
	}
	<-w.ctx.Done()
	err = context.Cause(w.ctx)
	if !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func (w *WsClient) UnSubscribe(arg *Arg, typ SvcType) error {
	if typ == Private && !w.Conns[typ].isLogin {
		return nil
	}
	return w.Send(typ, Op{
		Op:   "unsubscribe",
		Args: []*Arg{arg},
	})
}
