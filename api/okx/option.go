package okx

import (
	"net/http"
	"net/url"

	"github.com/kurosann/aqt-sdk/api"
)

func WithConfig(keyConfig OkxKeyConfig) api.Opt {
	return func(api api.IMarketApi) {
		if client, ok := api.(*ExchangeClient); ok {
			client.keyConfig = keyConfig
		}
	}
}
func WithProxy(proxy string) api.Opt {
	return func(api api.IMarketApi) {
		if client, ok := api.(ExchangeClient); ok {
			proxyURL := http.ProxyFromEnvironment
			if proxy != "" {
				parse, err := url.Parse(proxy)
				if err != nil {
					panic(err.Error())
				}
				proxyURL = http.ProxyURL(parse)
			}
			client.proxy = proxyURL
		}
	}
}
func WithEnv(env Destination) api.Opt {
	return func(api api.IMarketApi) {
		if client, ok := api.(ExchangeClient); ok {
			client.env = env
		}
	}
}
func WithEndpoints(urls map[Destination]map[SvcType]BaseURL) api.Opt {
	return func(api api.IMarketApi) {
		if client, ok := api.(ExchangeClient); ok {
			client.urls = urls
		}
	}
}
func UseOkxExchange(opts ...api.Opt) api.OptInfo {
	return api.OptInfo{
		Exchange: api.OkxExchange,
		Opts:     opts,
	}
}
