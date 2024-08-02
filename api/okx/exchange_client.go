package okx

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/kurosann/aqt-sdk/api"
)

type ExchangeClient struct {
	pc  *PublicClient
	bc  *BusinessClient
	pvc *PrivateClient

	urls      map[Destination]map[SvcType]BaseURL
	env       Destination
	keyConfig OkxKeyConfig
	proxy     func(req *http.Request) (*url.URL, error)
}

func NewExchangeClient(ctx context.Context, opts ...api.Opt) (*ExchangeClient, error) {
	client := ExchangeClient{
		proxy: http.ProxyFromEnvironment,
		urls:  DefaultWsUrls,
		env:   NormalServer,
	}
	for _, opt := range opts {
		opt(&client)
	}
	if client.keyConfig.Apikey == "" {
		return nil, errors.New("not config: Apikey or Secretkey or Passphrase")
	}
	client.pc = &PublicClient{BaseWsClient: NewBaseWsClient(ctx, Public, client.urls[client.env][Public], client.keyConfig, client.proxy)}
	client.bc = &BusinessClient{BaseWsClient: NewBaseWsClient(ctx, Business, client.urls[client.env][Business], client.keyConfig, client.proxy)}
	client.pvc = &PrivateClient{BaseWsClient: NewBaseWsClient(ctx, Private, client.urls[client.env][Private], client.keyConfig, client.proxy)}
	return &client
}

func (w *ExchangeClient) ReadMonitor(readMonitor func(arg Arg)) {
	w.pc.ReadMonitor = readMonitor
	w.bc.ReadMonitor = readMonitor
	w.pvc.ReadMonitor = readMonitor
}
func (w *ExchangeClient) SetLog(log api.ILogger) {
	w.pc.Log = log
	w.bc.Log = log
	w.pvc.Log = log
}
func (w *ExchangeClient) Close() {

}
