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
	ctx                 context.Context
	cancel              func()
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
	readMonitor         func(arg Arg)
	closeListen         func()
}
type WsClient struct {
	conns               map[SvcType]*WsClientConn
	ctx                 context.Context
	cancel              func()
	SubscribeCallback   func()
	UnSubscribeCallback func()
	connLock            sync.RWMutex
	loginLock           sync.RWMutex
	chanLock            sync.RWMutex
	subscribeLock       sync.RWMutex
	keyConfig           KeyConfig
	isLogin             map[SvcType]bool
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
		conns[svcType] = &WsClientConn{
			ctx:                 ctx,
			cancel:              cancel,
			typ:                 svcType,
			url:                 baseURL,
			SubscribeCallback:   func() {},
			UnSubscribeCallback: func() {},
			callbackMap:         make(map[string]func(*WsOriginResp)),
			log:                 DefaultLogger{},
			proxy:               proxyURL,
			readMonitor:         func(Arg) {},
			closeListen:         func() {},
		}
	}
	return &WsClient{
		ctx:         ctx,
		cancel:      cancel,
		keyConfig:   keyConfig,
		conns:       conns,
		isLogin:     make(map[SvcType]bool),
		log:         DefaultLogger{},
		closeListen: func() {},
		readMonitor: func(arg Arg) {},
		proxy:       proxyURL,
	}
}
func (w *WsClientConn) connect(ctx context.Context, url BaseURL) (*websocket.Conn, *http.Response, error) {
	dialer := websocket.DefaultDialer
	dialer.Proxy = w.proxy
	conn, rp, err := dialer.DialContext(ctx, string(url), nil)
	if err != nil {
		return nil, nil, err
	}
	go w.heartbeat(conn)
	return conn, rp, err
}

func (w *WsClientConn) heartbeat(conn *websocket.Conn) {
	timer := time.NewTicker(time.Second * 20)
	defer timer.Stop()
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-timer.C:
			conn.SetWriteDeadline(time.Now().Add(time.Second))
			err := conn.WriteMessage(websocket.TextMessage, []byte("ping"))
			if err != nil {
				w.log.Errorf(err.Error())
				w.Close()
				return
			}
		}
	}
}

func (w *WsClient) SetReadMonitor(readMonitor func(arg Arg)) {
	w.readMonitor = readMonitor
	for _, conn := range w.conns {
		conn.readMonitor = readMonitor
	}
}
func (w *WsClient) SetCloseListen(closeListen func()) {
	w.closeListen = closeListen
	for _, conn := range w.conns {
		conn.closeListen = closeListen
	}
}
func (w *WsClient) SetLog(log ILogger) {
	w.log = log
}
func (w *WsClient) Close() {

	for k, conn := range w.conns {
		conn.Close()
		delete(w.conns, k)
	}
	w.closeListen()
}
func (w *WsClientConn) Close() {
	if w.isClose {
		return
	}
	w.conn.Close()

	for k, _ := range w.callbackMap {
		delete(w.callbackMap, k)
	}
	w.isClose = true
	w.closeListen()
}
func (w *WsClientConn) lazyConnect() (*websocket.Conn, error) {
	if w.conn == nil || w.isClose {
		c, rp, err := w.connect(w.ctx, w.url)
		if err != nil {
			return nil, err
		}
		if rp.StatusCode != 101 {
			return nil, fmt.Errorf("connect response err: %v", rp)
		}
		// 监听连接关闭,进行资源回收
		c.SetCloseHandler(func(code int, text string) error {
			w.Close()
			return nil
		})
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
			w.Close()
			for {
				time.Sleep(time.Second * 3)
				_, err := w.lazyConnect()
				if err == nil {
					break
				}
				w.log.Errorf(err.Error())
			}
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
			w.Close()
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
	conn, err := w.conns[typ].lazyConnect()
	if err != nil {
		return err
	}
	bs, err := json.Marshal(req)
	if err != nil {
		return err
	}
	w.log.Infof(fmt.Sprintf("%s:send %s", typ, string(bs)))
	w.connLock.Lock()
	defer w.connLock.Unlock()
	if w.conns[typ].isClose {
		return errors.New("conn is close")
	}
	conn.SetWriteDeadline(time.Now().Add(time.Second * 3))
	return conn.WriteMessage(websocket.TextMessage, bs)
}

func (w *WsClient) login(typ SvcType) error {
	w.loginLock.Lock()
	defer w.loginLock.Unlock()

	if w.isLogin[typ] {
		return nil
	}
	c := make(chan *WsOriginResp, 1)
	if err := w.Login(typ, func(resp *WsOriginResp) {
		c <- resp
	}); err != nil {
		return err
	}
	resp := <-c
	if resp.Event == "error" || !w.isLogin[typ] {
		return errors.New(resp.Msg)
	}
	return nil
}
func Subscribe[T any](w *WsClient, arg *Arg, typ SvcType, callback func(resp *WsResp[T]), needLogin bool) error {
	w.subscribeLock.Lock()
	defer w.subscribeLock.Unlock()

	if needLogin && !w.isLogin[typ] {
		if err := w.login(typ); err != nil {
			return err
		}
	}
	ctx, cancel := context.WithCancel(w.ctx)
	w.conns[typ].RegCallback(arg.Key(), func(resp *WsOriginResp) {
		if resp.Event == "subscribe" {
			cancel()
			return
		}
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
	})
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				if err := w.Send(typ, Op{
					Op:   "subscribe",
					Args: []*Arg{arg},
				}); err != nil {
					return
				}
			}
		}
	}()
	<-ctx.Done()
	return nil
}

func (w *WsClient) UnSubscribe(arg *Arg, typ SvcType) error {
	if typ == Private && !w.isLogin[typ] {
		return nil
	}
	return w.Send(typ, Op{
		Op:   "unsubscribe",
		Args: []*Arg{arg},
	})
}
