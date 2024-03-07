package okx

import (
	"context"
	"crypto/md5"
	"encoding/hex"
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
	chanMap             map[string]chan *WsResp
	keyConfig           KeyConfig
	isLogin             bool
	log                 Log
	CloseListen         func()
	ReadMonitor         func(arg Arg)
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
		ctx:       ctx,
		cancel:    cancel,
		urls:      urls[env],
		keyConfig: keyConfig,
		conns:     map[SvcType]*websocket.Conn{},
		chanMap:   make(map[string]chan *WsResp, 1),
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
	go w.heartbeat(conn)
	return conn, rp, err
}

func (w *WsClient) heartbeat(conn *websocket.Conn) {
	timer := time.NewTimer(time.Second * 20)
	for {
		<-timer.C
		err := conn.WriteMessage(websocket.TextMessage, []byte("ping"))
		var closeError *websocket.CloseError
		if errors.As(err, &closeError) {
			w.CloseListen()
			return
		}
	}
}

func (w *WsClient) SetLog(log Log) {
	w.log = log
}
func (w *WsClient) Close() {
	if w.connLock.TryLock() {
		return
	}
	defer w.connLock.Unlock()

	for k, conn := range w.conns {
		_ = conn.Close()

		delete(w.conns, k)
	}
	for k, c := range w.chanMap {
		close(c)
		delete(w.chanMap, k)
	}
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
			w.CloseListen()
			return nil
		})
		conn = c
		w.conns[typ] = conn

		go w.process(typ, conn)
	}
	return conn, nil
}
func (w *WsClient) push(typ SvcType, channel string, resp *WsResp) {
	w.connLock.RLock()
	defer w.connLock.RUnlock()
	if ch, ok := w.chanMap[channel]; ok {
		select {
		case ch <- resp:
		case <-time.After(time.Second * 3):
			w.log.Warn(fmt.Sprintf("%s:insert channel timeout, ignore data: %+v", typ, resp))
		}
	} else {
		w.log.Warn(fmt.Sprintf("%s:not found channel, ignore data: %+v", typ, resp))
	}
}

func (w *WsClient) RegCh(channel string, c chan *WsResp) (chan *WsResp, bool) {
	w.chanLock.Lock()
	defer w.chanLock.Unlock()

	if ch, ok := w.chanMap[channel]; ok {
		return ch, true
	} else {
		w.chanMap[channel] = c
	}
	return w.chanMap[channel], false
}
func (w *WsClient) GetCh(channel string) (chan *WsResp, bool) {
	w.chanLock.RLock()
	defer w.chanLock.RUnlock()
	if respCh, ok := w.chanMap[channel]; ok {
		return respCh, ok
	}
	return nil, false
}

func (w *WsClient) UnRegCh(channel string) {
	w.chanLock.Lock()
	defer w.chanLock.Unlock()

	if respCh, ok := w.chanMap[channel]; ok {
		close(respCh)
		delete(w.chanMap, channel)
	}
}
func (w *WsClient) process(typ SvcType, conn *websocket.Conn) {
	defer func() {
		_ = recover()
	}()
	for {
		mtp, bs, err := conn.ReadMessage()
		var closeError *websocket.CloseError
		if errors.As(err, &closeError) {
			w.CloseListen()
			return
		}
		if mtp != websocket.TextMessage {
			continue
		}
		v := &WsResp{}
		err = json.Unmarshal(bs, v)
		if err != nil {
			continue
		}
		w.ReadMonitor(v.Arg)
		bs, err = json.Marshal(v.Arg)
		hash := md5.New()
		hash.Write(bs)
		channel := hex.EncodeToString(hash.Sum(nil))
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
			w.isLogin = true
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
		w.push(typ, channel, v)
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
	return conn.WriteMessage(websocket.TextMessage, bs)
}

func (w *WsClient) Subscribe(arg *Arg, typ SvcType, needLogin bool) (<-chan *WsResp, error) {
	if needLogin && !w.isLogin {
		w.loginLock.Lock()
		defer w.loginLock.Unlock()
		ch, err := w.Login(typ)
		if err != nil {
			return nil, err
		}
		resp := <-ch
		if resp.Event == "error" || !w.isLogin {
			return nil, errors.New(resp.Msg)
		}
	}
	bs, err := json.Marshal(arg)
	if err != nil {
		return nil, err
	}
	hash := md5.New()
	hash.Write(bs)
	hex.EncodeToString(hash.Sum(nil))
	respCh := make(chan *WsResp)
	if ch, isExist := w.RegCh(hex.EncodeToString(hash.Sum(nil)), respCh); isExist {
		respCh = ch
	}
	if err := w.Send(typ, Op{
		Op:   "subscribe",
		Args: []*Arg{arg},
	}); err != nil {
		return nil, err
	}
loop:
	for {
		select {
		case <-time.After(time.Second * 3):
			_ = w.Send(typ, Op{
				Op:   "subscribe",
				Args: []*Arg{arg},
			})
		case resp, ok := <-respCh:
			if !ok {
				time.Sleep(time.Second)
				_ = w.Send(typ, Op{
					Op:   "subscribe",
					Args: []*Arg{arg},
				})
			}
			if resp.Event == "subscribe" {
				break loop
			}
		}
	}
	return respCh, nil
}

func (w *WsClient) UnSubscribe(arg *Arg, typ SvcType) error {
	if typ == Private && !w.isLogin {
		return nil
	}
	return w.Send(typ, Op{
		Op:   "unsubscribe",
		Args: []*Arg{arg},
	})
}
