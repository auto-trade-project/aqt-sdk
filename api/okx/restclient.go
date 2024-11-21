package okx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RestClient struct {
	baseUrl   BaseURL
	ctx       context.Context
	client    *http.Client
	cancel    context.CancelFunc
	keyConfig OkxKeyConfig
	isTest    bool
}

func NewRestClient(ctx context.Context, keyConfig OkxKeyConfig, env Destination, urls map[Destination]BaseURL, proxy func(req *http.Request) (*url.URL, error)) RestClient {
	ctx, cancel := context.WithCancel(ctx)
	baseUrl, ok := urls[env]
	if !ok {
		panic("not support env")
	}
	proxyURL := http.ProxyFromEnvironment
	if proxy != nil {
		proxyURL = proxy
	}
	return RestClient{
		ctx:       ctx,
		cancel:    cancel,
		baseUrl:   baseUrl,
		keyConfig: keyConfig,
		isTest:    env == TestServer,
		client: &http.Client{
			Transport: &http.Transport{
				Proxy: proxyURL,
			},
			Timeout: 30 * time.Second,
		}}
}

func Get[T any](c *RestClient, ctx context.Context, url string, params interface{}) (*Resp[T], error) {
	return Do[T](c, c.MakeRequest(ctx, http.MethodGet, url, params))
}
func Post[T any](c *RestClient, ctx context.Context, url string, params interface{}) (*Resp[T], error) {
	return Do[T](c, c.MakeRequest(ctx, http.MethodPost, url, params))
}
func Do[T any](c *RestClient, req *http.Request) (*Resp[T], error) {
	rp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	bs, err := io.ReadAll(rp.Body)
	if err != nil {
		return nil, err
	}
	if rp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%v %v, statusCode is %v, msg: %v", req.Method, req.URL.String(), rp.StatusCode, string(bs))
	}
	t, err := Unmarshal[Resp[T]](bs)
	if err != nil {
		return nil, err
	}
	if t.Code != "0" {
		return nil, fmt.Errorf(t.Msg)
	}
	return t, nil
}
func (c *RestClient) MakeRequest(ctx context.Context, method, url string, params interface{}) *http.Request {
	bs, _ := json.Marshal(params)
	uri := ""
	if method == http.MethodGet {
		uri += makeUri(bs)
		bs = nil
	}
	req, _ := http.NewRequestWithContext(ctx, method, string(c.baseUrl)+url+uri, bytes.NewReader(bs))
	header := c.keyConfig.MakeHeader(method, url+uri, bs)
	if c.isTest {
		header["x-simulated-trading"] = []string{"1"}
	}
	req.Header = header
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}
func makeUri(bs []byte) (uri string) {
	query := map[string]interface{}{}
	_ = json.Unmarshal(bs, &query)
	if len(query) != 0 {
		var fields []string
		for k, item := range query {
			if item != "" {
				fields = append(fields, fmt.Sprintf("%v=%v", k, item))
			}
		}
		if len(fields) != 0 {
			uri += "?"
			uri += strings.Join(fields, "&")
		}
	}
	return uri
}
