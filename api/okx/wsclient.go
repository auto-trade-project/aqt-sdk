package okx

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
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
	ctx        context.Context
	cancel     func()
	urls       map[SvcType]BaseURL
	conns      map[SvcType]*websocket.Conn
	apikey     string
	secretkey  string
	passphrase string
	l          sync.RWMutex
	chanMap    map[string]chan *WsResp
	keyConfig  KeyConfig
}

func NewWsClient(ctx context.Context, keyConfig KeyConfig, env Destination) *WsClient {
	urls := map[SvcType]BaseURL{}
	switch env {
	case NormalServer:
		urls[Public], urls[Private], urls[Business] = PublicWsURL, PrivateWsURL, BusinessWsURL
	case AwsServer:
		urls[Public], urls[Private], urls[Business] = AwsPublicWsURL, AwsPrivateWsURL, AwsBusinessWsURL
	case TestServer:
		urls[Public], urls[Private], urls[Business] = TestPublicWsURL, TestPrivateWsURL, TestBusinessWsURL
	default:
		panic("not support env")
	}
	ctx, cancel := context.WithCancel(ctx)
	return &WsClient{
		ctx:       ctx,
		cancel:    cancel,
		urls:      urls,
		keyConfig: keyConfig,
		conns:     map[SvcType]*websocket.Conn{},
		chanMap:   make(map[string]chan *WsResp),
	}
}
func connect(ctx context.Context, url BaseURL) (*websocket.Conn, *http.Response, error) {
	conn, rp, err := websocket.DefaultDialer.DialContext(ctx, string(url), nil)
	go process(conn)
	return conn, rp, err
}

func process(conn *websocket.Conn) {
	timer := time.NewTimer(time.Second * 20)
	for {
		select {
		case <-timer.C:
			err := conn.WriteMessage(websocket.TextMessage, []byte("ping"))
			if err != nil {
				return
			}
		}
	}
}

func (w *WsClient) lazyConnect(typ SvcType) (*websocket.Conn, error) {
	conn, ok := w.conns[typ]
	if !ok {
		c, rp, err := connect(w.ctx, w.urls[typ])
		if err != nil {
			return nil, err
		}
		if rp.StatusCode != 101 {
			return nil, fmt.Errorf("connect response err: %v", rp)
		}
		conn = c
		w.conns[typ] = conn
		go w.process(conn)
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

func (w *WsClient) RegCh(channel string, f chan *WsResp) (chan *WsResp, bool) {
	w.l.Lock()
	defer w.l.Unlock()

	if ch, ok := w.chanMap[channel]; ok {
		return ch, true
	} else {
		w.chanMap[channel] = f
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
func (w *WsClient) process(conn *websocket.Conn) {
	for {
		v := &WsResp{}
		err := conn.ReadJSON(v)
		if err != nil {
			return
		}
		w.push(v.Arg.Channel+v.Arg.InstId, v)
	}
}

func (w *WsClient) Send(typ SvcType, req any) error {
	conn, err := w.lazyConnect(typ)
	if err != nil {
		return err
	}
	return conn.WriteJSON(req)
}

// Subscribe channel要与 UnSubscribe 的channel对应上 否则无法关闭chan
func (w *WsClient) Subscribe(arg Arg, typ SvcType) (<-chan *WsResp, error) {
	err := w.Send(typ, Op{
		Op:   "subscribe",
		Args: []Arg{arg},
	})
	if err != nil {
		return nil, err
	}
	valueOfArg := reflect.ValueOf(arg)
	channel := ""
	for i := 0; i < valueOfArg.NumField(); i++ {
		field := valueOfArg.Field(i)
		channel += field.String()
	}
	respCh := make(chan *WsResp)
	if respCh, ok := w.RegCh(channel, respCh); ok {
		return respCh, nil
	}
	return respCh, nil
}

// UnSubscribe channel要与 Subscribe 的channel对应上 否则无法关闭chan
func (w *WsClient) UnSubscribe(arg Arg, typ SvcType) error {
	valueOfArg := reflect.ValueOf(arg)
	channel := ""
	for i := 0; i < valueOfArg.NumMethod(); i++ {
		field := valueOfArg.Field(i)
		channel += field.String()
	}
	w.UnRegCh(channel)
	return w.Send(typ, Op{
		Op:   "unsubscribe",
		Args: []Arg{arg},
	})
}
