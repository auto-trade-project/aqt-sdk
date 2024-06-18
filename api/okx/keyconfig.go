package okx

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type KeyConfig struct {
	Apikey     string
	Secretkey  string
	Passphrase string
}

func (c KeyConfig) MakeHeader(method, requestPath string, body []byte) http.Header {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.999Z")
	sign := c.MakeSign(now, method, requestPath, body)
	return map[string][]string{
		"OK-ACCESS-KEY":        {c.Apikey},
		"OK-ACCESS-SIGN":       {sign},
		"OK-ACCESS-TIMESTAMP":  {now},
		"OK-ACCESS-PASSPHRASE": {c.Passphrase},
	}
}

func (c KeyConfig) MakeWsSign() map[string]string {
	now := fmt.Sprint(time.Now().UTC().Unix())
	sign := c.MakeSign(now, http.MethodGet, "/users/self/verify", []byte(""))
	return map[string]string{
		"apiKey":     c.Apikey,
		"sign":       sign,
		"timestamp":  now,
		"passphrase": c.Passphrase,
	}
}
func (c KeyConfig) MakeSign(now, method, requestPath string, body []byte) (sign string) {
	method = strings.ToUpper(method)
	hash := hmac.New(sha256.New, []byte(c.Secretkey))
	hash.Write(append([]byte(now+method+requestPath), body...))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}
