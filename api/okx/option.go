package okx

import (
	"net/http"
	"net/url"

	"github.com/kurosann/aqt-sdk/api"
)

func WithConfig(keyConfig OkxKeyConfig) api.Opt {
	return func(api api.IMarketClient) {
		if client, ok := api.(*ExchangeClient); ok {
			client.keyConfig = keyConfig
		}
	}
}
func WithProxy(proxy string) api.Opt {
	return func(api api.IMarketClient) {
		if proxy != "" {
			if client, ok := api.(*ExchangeClient); ok {
				proxyURL := http.ProxyFromEnvironment
				parse, err := url.Parse(proxy)
				if err != nil {
					panic(err.Error())
				}
				proxyURL = http.ProxyURL(parse)
				client.proxy = proxyURL
			}
		}
	}
}
func WithEnv(env Destination) api.Opt {
	return func(api api.IMarketClient) {
		if client, ok := api.(*ExchangeClient); ok {
			client.env = env
		}
	}
}
func WithWsEndpoints(urls map[Destination]map[SvcType]BaseURL) api.Opt {
	return func(api api.IMarketClient) {
		if client, ok := api.(*ExchangeClient); ok {
			client.wsUrls = urls
		}
	}
}
func WithRestEndpoints(urls map[Destination]BaseURL) api.Opt {
	return func(api api.IMarketClient) {
		if client, ok := api.(*ExchangeClient); ok {
			client.restUrls = urls
		}
	}
}
func UseOkxExchange(opts ...api.Opt) api.OptInfo {
	return api.NewOptInfo(api.OkxExchange, opts...)
}
