package okx

import (
	"context"
	"net/http"
	"net/url"

	"github.com/kurosann/aqt-sdk/api/common"
)

type ExchangeClient struct {
	*PublicClient
	*BusinessClient
	*PrivateClient
}

func NewWsClient(ctx context.Context, keyConfig common.IKeyConfig, env common.Destination, proxy ...string) *ExchangeClient {
	return NewWsClientWithCustom(ctx, keyConfig, env, common.DefaultWsUrls, proxy...)
}

func NewWsClientWithCustom(ctx context.Context, keyConfig common.IKeyConfig, env common.Destination, urls map[common.Destination]map[common.SvcType]common.BaseURL, proxy ...string) *ExchangeClient {
	proxyURL := http.ProxyFromEnvironment
	if len(proxy) != 0 && proxy[0] != "" {
		parse, err := url.Parse(proxy[0])
		if err != nil {
			panic(err.Error())
		}
		proxyURL = http.ProxyURL(parse)
	}
	pc := &PublicClient{WsClient: common.NewBaseWsClient(ctx, common.Public, urls[env][common.Public], keyConfig, proxyURL)}
	bc := &BusinessClient{WsClient: common.NewBaseWsClient(ctx, common.Business, urls[env][common.Business], keyConfig, proxyURL)}
	pvc := &PrivateClient{WsClient: common.NewBaseWsClient(ctx, common.Private, urls[env][common.Private], keyConfig, proxyURL)}
	return &ExchangeClient{
		pc,
		bc,
		pvc,
	}
}

func (w *ExchangeClient) SetReadMonitor(readMonitor func(arg common.Arg)) {
	w.PublicClient.ReadMonitor = readMonitor
	w.BusinessClient.ReadMonitor = readMonitor
	w.PrivateClient.ReadMonitor = readMonitor
}
func (w *ExchangeClient) SetLog(log common.ILogger) {
	w.PublicClient.Log = log
	w.BusinessClient.Log = log
	w.PrivateClient.Log = log
}
func (w *ExchangeClient) Close() {

}
