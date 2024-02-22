package okx

import (
	"context"
	"fmt"
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
			err := conn.WriteMessage(websocket.TextMessage, []byte("pong"))
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
		ch <- resp
	}
}

func (w *WsClient) RegCh(channel string, f chan *WsResp) {
	w.l.Lock()
	defer w.l.Unlock()

	w.chanMap[channel] = f
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
		w.push(v.Arg.Channel, v)
	}
}

func (w *WsClient) Send(channel string, typ SvcType, req any) (<-chan *WsResp, error) {
	conn, err := w.lazyConnect(typ)
	if err != nil {
		return nil, err
	}
	respCh := make(chan *WsResp)
	w.RegCh(channel, respCh)
	defer func() {
		if err != nil {
			w.UnRegCh(channel)
		}
	}()
	err = conn.WriteJSON(req)
	return respCh, err
}

func (w *WsClient) Subscribe(channel, InstId string, typ SvcType) (<-chan *WsResp, error) {
	return w.Send(channel, typ, makeSubscribeOp(channel, InstId))
}
func (w *WsClient) UnSubscribe(channel, InstId string, typ SvcType) (<-chan *WsResp, error) {
	return w.Send(channel, typ, makeUnSubscribeOp(channel, InstId))
}
