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
	l                   sync.RWMutex
	chanMap             map[string]chan *WsResp
	keyConfig           KeyConfig
	isLogin             bool
}

func NewWsClient(ctx context.Context, keyConfig KeyConfig, env Destination) *WsClient {
	return NewWsClientWithCustom(ctx, keyConfig, env, DefaultWsUrls)
}

func NewWsClientWithCustom(ctx context.Context, keyConfig KeyConfig, env Destination, urls map[Destination]map[SvcType]BaseURL) *WsClient {
	ctx, cancel := context.WithCancel(ctx)
	return &WsClient{
		ctx:       ctx,
		cancel:    cancel,
		urls:      urls[env],
		keyConfig: keyConfig,
		conns:     map[SvcType]*websocket.Conn{},
		chanMap:   make(map[string]chan *WsResp),
	}
}
func connect(ctx context.Context, url BaseURL) (*websocket.Conn, *http.Response, error) {
	conn, rp, err := websocket.DefaultDialer.DialContext(ctx, string(url), nil)
	go heartbeat(conn)
	return conn, rp, err
}

func heartbeat(conn *websocket.Conn) {
	timer := time.NewTimer(time.Second * 20)
	for {
		<-timer.C
		err := conn.WriteMessage(websocket.TextMessage, []byte("ping"))
		if err != nil {
			return
		}
	}
}

func (w *WsClient) lazyConnect(typ SvcType) (*websocket.Conn, error) {
	w.l.RLock()
	conn, ok := w.conns[typ]
	w.l.RUnlock()
	if !ok {
		c, rp, err := connect(w.ctx, w.urls[typ])
		if err != nil {
			return nil, err
		}
		if rp.StatusCode != 101 {
			return nil, fmt.Errorf("connect response err: %v", rp)
		}
		// 监听连接关闭,进行资源回收
		c.SetCloseHandler(func(code int, text string) error {
			w.l.Lock()
			defer w.l.Unlock()

			delete(w.conns, typ)
			return nil
		})
		conn = c
		w.l.Lock()
		defer w.l.Unlock()
		w.conns[typ] = conn

		go w.process(typ, conn)
	}
	return conn, nil
}
func (w *WsClient) push(channel string, resp *WsResp) {
	w.l.RLock()
	defer w.l.RUnlock()
	if ch, ok := w.chanMap[channel]; ok {
		select {
		case ch <- resp:
		case <-time.After(time.Second * 3):
			fmt.Printf("ignore data: %v\n", resp)
		}
	} else {
		fmt.Printf("ignore data: %v\n", resp)
	}
}

func (w *WsClient) RegCh(channel string, c chan *WsResp) (chan *WsResp, bool) {
	w.l.Lock()
	defer w.l.Unlock()

	if ch, ok := w.chanMap[channel]; ok {
		return ch, true
	} else {
		w.chanMap[channel] = c
	}
	return w.chanMap[channel], false
}
func (w *WsClient) GetCh(channel string) (chan *WsResp, bool) {
	w.l.Lock()
	defer w.l.Unlock()
	if respCh, ok := w.chanMap[channel]; ok {
		return respCh, ok
	}
	return nil, false
}

func (w *WsClient) UnRegCh(channel string) {
	w.l.Lock()
	defer w.l.Unlock()

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
		v := &WsResp{}
		err := conn.ReadJSON(v)
		if err != nil {
			continue
		}

		bs, _ := json.Marshal(v.Arg)
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
			continue
		} else if v.Event == "login" {
			log.Println("login success")
			w.isLogin = true
			w.push("login", v)
			w.UnRegCh("login")
			continue
		} else if v.Event == "error" {
			if v.Code == "60009" {
				w.push("login", v)
			}
			log.Printf("Service net '%s' read ws err: %v\n", typ, v)
			continue
		}
		if v.Data == nil {
			log.Printf("ignore data: %v\n", v)
			continue
		}
		w.push(channel, v)
	}
}

func (w *WsClient) Send(typ SvcType, req any) error {
	conn, err := w.lazyConnect(typ)
	if err != nil {
		return err
	}
	return conn.WriteJSON(req)
}

func (w *WsClient) Subscribe(arg *Arg, typ SvcType) (<-chan *WsResp, error) {
	if typ == Private && !w.isLogin {
		ch, err := w.Login()
		if err != nil {
			return nil, err
		}
		resp := <-ch
		if resp.Event == "error" || !w.isLogin {
			return nil, errors.New(resp.Msg)
		}
	}
	if err := w.Send(typ, Op{
		Op:   "subscribe",
		Args: []*Arg{arg},
	}); err != nil {
		return nil, err
	}
	bs, err := json.Marshal(arg)
	if err != nil {
		return nil, err
	}
	hash := md5.New()
	hash.Write(bs)
	hex.EncodeToString(hash.Sum(nil))
	respCh := make(chan *WsResp)
	if respCh, isExist := w.RegCh(hex.EncodeToString(hash.Sum(nil)), respCh); isExist {
		return respCh, nil
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
