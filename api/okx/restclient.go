package okx

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type RestClient struct {
	baseUrl   BaseURL
	ctx       context.Context
	client    *http.Client
	cancel    context.CancelFunc
	keyConfig KeyConfig
	isTest    bool
}

func NewRestClient(ctx context.Context, keyConfig KeyConfig, env Destination) *RestClient {
	var baseUrl BaseURL
	switch env {
	case NormalServer:
		baseUrl = RestURL
	case AwsServer:
		baseUrl = AwsRestURL
	case TestServer:
		baseUrl = TestRestURL
	default:
		panic("not support env")
	}
	ctx, cancel := context.WithCancel(ctx)
	return &RestClient{
		ctx:       ctx,
		cancel:    cancel,
		baseUrl:   baseUrl,
		keyConfig: keyConfig,
		isTest:    env == TestServer,
		client: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
			Timeout: 30 * time.Second,
		}}
}

func (c KeyConfig) makeHeader(method, requestPath string, body []byte) http.Header {
	sign, now := c.makeSign(method, requestPath, body)
	return map[string][]string{
		"OK-ACCESS-KEY":        {c.Apikey},
		"OK-ACCESS-SIGN":       {sign},
		"OK-ACCESS-TIMESTAMP":  {now},
		"OK-ACCESS-PASSPHRASE": {c.Passphrase},
	}
}

func (c KeyConfig) makeWsSign() map[string]string {
	sign, now := c.makeSign("GET", "/users/self/verify", []byte(""))
	return map[string]string{
		"OK-ACCESS-KEY":        c.Apikey,
		"OK-ACCESS-SIGN":       sign,
		"OK-ACCESS-TIMESTAMP":  now,
		"OK-ACCESS-PASSPHRASE": c.Passphrase,
	}
}
func (c KeyConfig) makeSign(method, requestPath string, body []byte) (sign string, now string) {
	method = strings.ToUpper(method)
	now = time.Now().Format(time.RFC3339)
	hash := hmac.New(sha256.New, []byte(c.Secretkey))
	hash.Write(append([]byte(now+method+requestPath), body...))
	return hex.EncodeToString(hash.Sum(nil)), now
}
func (c RestClient) makeGet(ctx context.Context, url string, params interface{}) *http.Request {
	return c.makeRequest(ctx, "GET", url, params)
}
func (c RestClient) makePost(ctx context.Context, url string, params interface{}) *http.Request {
	return c.makeRequest(ctx, "POST", url, params)
}
func (c RestClient) makeRequest(ctx context.Context, method, url string, params interface{}) *http.Request {
	bs, _ := json.Marshal(params)
	uri := ""
	if method == http.MethodGet {
		uri += makeUri(bs)
		bs = nil
	}
	req, _ := http.NewRequestWithContext(ctx, method, string(c.baseUrl)+url+uri, bytes.NewReader(bs))
	header := c.keyConfig.makeHeader(method, url, bs)
	if c.isTest {
		header["x-simulated-trading"] = []string{"1"}
	}
	req.Header = header
	return req
}
func makeUri(bs []byte) (uri string) {
	query := map[string]interface{}{}
	_ = json.Unmarshal(bs, &query)
	if len(query) != 0 {
		uri += "?"
		var fields []string
		for k, item := range query {
			fields = append(fields, fmt.Sprintf("%v=%v", k, item))
		}
		uri += strings.Join(fields, "&")
	}
	return uri
}
