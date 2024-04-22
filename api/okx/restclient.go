package okx

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type RestClient struct {
	baseUrl   BaseURL
	ctx       context.Context
	client    *http.Client
	cancel    context.CancelFunc
	keyConfig KeyConfig
	isTest    bool
	limitReq  int
	locker    sync.RWMutex
}

func NewRestClient(ctx context.Context, keyConfig KeyConfig, env Destination, proxy ...string) *RestClient {
	return NewRestClientWithCustom(ctx, keyConfig, env, DefaultRestUrl, proxy...)
}

func NewRestClientWithCustom(ctx context.Context, keyConfig KeyConfig, env Destination, urls map[Destination]BaseURL, proxy ...string) *RestClient {
	ctx, cancel := context.WithCancel(ctx)
	baseUrl, ok := urls[env]
	if !ok {
		panic("not support env")
	}
	proxyURL := http.ProxyFromEnvironment
	if len(proxy) != 0 && proxy[0] != "" {
		parse, err := url.Parse(proxy[0])
		if err != nil {
			panic(err.Error())
		}
		proxyURL = http.ProxyURL(parse)
	}
	return &RestClient{
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

func (c KeyConfig) makeHeader(method, requestPath string, body []byte) http.Header {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.999Z")
	sign := c.makeSign(now, method, requestPath, body)
	return map[string][]string{
		"OK-ACCESS-KEY":        {c.Apikey},
		"OK-ACCESS-SIGN":       {sign},
		"OK-ACCESS-TIMESTAMP":  {now},
		"OK-ACCESS-PASSPHRASE": {c.Passphrase},
	}
}

func (c KeyConfig) makeWsSign() map[string]string {
	now := fmt.Sprint(time.Now().UTC().Unix())
	sign := c.makeSign(now, http.MethodGet, "/users/self/verify", []byte(""))
	return map[string]string{
		"apiKey":     c.Apikey,
		"sign":       sign,
		"timestamp":  now,
		"passphrase": c.Passphrase,
	}
}
func (c KeyConfig) makeSign(now, method, requestPath string, body []byte) (sign string) {
	method = strings.ToUpper(method)
	hash := hmac.New(sha256.New, []byte(c.Secretkey))
	hash.Write(append([]byte(now+method+requestPath), body...))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
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
	t, err := unmarshal[Resp[T]](bs)
	if err != nil {
		return nil, err
	}
	if t.Code != "0" {
		return nil, fmt.Errorf(t.Msg)
	}
	return t, nil
}
func (c RestClient) MakeRequest(ctx context.Context, method, url string, params interface{}) *http.Request {
	bs, _ := json.Marshal(params)
	uri := ""
	if method == http.MethodGet {
		uri += makeUri(bs)
		bs = nil
	}
	req, _ := http.NewRequestWithContext(ctx, method, string(c.baseUrl)+url+uri, bytes.NewReader(bs))
	header := c.keyConfig.makeHeader(method, url+uri, bs)
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
