package okx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
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

type WsClient struct {
	ctx                 context.Context
	cancel              func()
	urls                map[SvcType]BaseURL
	conns               map[SvcType]*websocket.Conn
	SubscribeCallback   func()
	UnSubscribeCallback func()
	connLock            sync.RWMutex
	loginLock           sync.RWMutex
	chanLock            sync.RWMutex
	subscribeLock       sync.RWMutex
	callbackMap         map[string]func(*WsOriginResp)
	keyConfig           KeyConfig
	isLogin             map[SvcType]bool
	log                 Log
	CloseListen         func()
	ReadMonitor         func(arg Arg)
	isClose             bool
}
type Log struct {
	Info  func(msg string)
	Warn  func(msg string)
	Error func(msg string)
}

func NewWsClient(ctx context.Context, keyConfig KeyConfig, env Destination) *WsClient {
	return NewWsClientWithCustom(ctx, keyConfig, env, DefaultWsUrls)
}

func NewWsClientWithCustom(ctx context.Context, keyConfig KeyConfig, env Destination, urls map[Destination]map[SvcType]BaseURL) *WsClient {
	ctx, cancel := context.WithCancel(ctx)
	l := func(msg string) {
		log.Println(msg)
	}
	return &WsClient{
		ctx:         ctx,
		cancel:      cancel,
		urls:        urls[env],
		keyConfig:   keyConfig,
		conns:       map[SvcType]*websocket.Conn{},
		callbackMap: make(map[string]func(resp *WsOriginResp)),
		isLogin:     make(map[SvcType]bool),
		log: Log{
			Info:  l,
			Warn:  l,
			Error: l,
		},
		CloseListen: func() {},
		ReadMonitor: func(arg Arg) {},
	}
}
func (w *WsClient) connect(ctx context.Context, url BaseURL) (*websocket.Conn, *http.Response, error) {
	conn, rp, err := websocket.DefaultDialer.DialContext(ctx, string(url), nil)
	if err != nil {
		return nil, nil, err
	}
	go w.heartbeat(conn)
	return conn, rp, err
}

func (w *WsClient) heartbeat(conn *websocket.Conn) {
	timer := time.NewTicker(time.Second * 20)
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-timer.C:
			conn.SetWriteDeadline(time.Now().Add(time.Second))
			err := conn.WriteMessage(websocket.TextMessage, []byte("ping"))
			if err != nil {
				w.log.Error(err.Error())
				w.Close()
				return
			}
		}
	}
}

func (w *WsClient) SetLog(log Log) {
	w.log = log
}
func (w *WsClient) Close() {
	w.connLock.Lock()
	defer w.connLock.Unlock()
	if w.isClose {
		return
	}

	for k, conn := range w.conns {
		_ = conn.Close()

		delete(w.conns, k)
	}
	for k, _ := range w.callbackMap {
		delete(w.callbackMap, k)
	}
	w.CloseListen()
	w.isClose = true
}
func (w *WsClient) lazyConnect(typ SvcType) (*websocket.Conn, error) {
	w.connLock.Lock()
	defer w.connLock.Unlock()
	conn, ok := w.conns[typ]
	if !ok {
		c, rp, err := w.connect(w.ctx, w.urls[typ])
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
		conn = c
		w.conns[typ] = conn

		go w.process(typ, conn)
	}
	return conn, nil
}
func (w *WsClient) push(typ SvcType, channel string, resp *WsOriginResp) {
	if callback, ok := w.GetCallback(channel); ok {
		callback(resp)
	} else {
		w.log.Warn(fmt.Sprintf("%s:not found channel, ignore data: %+v", typ, resp))
	}
}

func (w *WsClient) RegCallback(channel string, c func(*WsOriginResp)) {
	w.chanLock.Lock()
	defer w.chanLock.Unlock()

	w.callbackMap[channel] = c
}
func (w *WsClient) GetCallback(channel string) (func(*WsOriginResp), bool) {
	w.chanLock.RLock()
	defer w.chanLock.RUnlock()
	if respCh, ok := w.callbackMap[channel]; ok {
		return respCh, ok
	}
	return nil, false
}

func (w *WsClient) UnRegCh(channel string) {
	w.chanLock.Lock()
	defer w.chanLock.Unlock()

	if _, ok := w.callbackMap[channel]; ok {
		delete(w.callbackMap, channel)
	}
}
func (w *WsClient) process(typ SvcType, conn *websocket.Conn) {
	defer func() {
		if err := recover(); err != nil {
			w.log.Error(fmt.Sprintf("%v", err))
		}
	}()
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}
		mtp, bs, err := conn.ReadMessage()
		if err != nil {
			w.log.Error(err.Error())
			w.Close()
			return
		}
		if mtp != websocket.TextMessage {
			continue
		}
		v := &WsOriginResp{}
		err = json.Unmarshal(bs, v)
		if err != nil {
			continue
		}
		w.ReadMonitor(v.Arg)
		if v.Event == "unsubscribe" {
			if w.UnSubscribeCallback != nil {
				w.UnSubscribeCallback()
			}
			continue
		} else if v.Event == "subscribe" {
			if w.SubscribeCallback != nil {
				w.SubscribeCallback()
			}
			w.log.Info(fmt.Sprintf("%s:%+v subscribe success", typ, v.Arg))
		} else if v.Event == "login" {
			w.log.Info(fmt.Sprintf("%s:login success", typ))
			w.isLogin[typ] = true
			w.push(typ, "login", v)
			continue
		} else if v.Event == "error" {
			if v.Code == "60009" {
				w.push(typ, "login", v)
			}
			w.log.Error(fmt.Sprintf("%s:read ws err: %v", typ, v))
			continue
		}
		//if v.Data == nil {
		//	w.log.Warn(fmt.Sprintf("%s:ignore data: %+v", typ, v))
		//	continue
		//}
		w.push(typ, v.Arg.Channel, v)
	}
}

func (w *WsClient) Send(typ SvcType, req any) error {
	conn, err := w.lazyConnect(typ)
	if err != nil {
		return err
	}
	bs, err := json.Marshal(req)
	if err != nil {
		return err
	}
	w.log.Info(fmt.Sprintf("%s:send %s", typ, string(bs)))
	w.connLock.Lock()
	defer w.connLock.Unlock()
	if w.isClose {
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
	w.RegCallback(arg.Channel, func(resp *WsOriginResp) {
		if resp.Event == "subscribe" {
			cancel()
			return
		}
		var t []T
		err := json.Unmarshal(resp.Data, &t)
		if err != nil {
			w.log.Error(err.Error())
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
