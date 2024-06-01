package ws

import (
	"net/http"
	"net/url"
	"time"
)

type Option func(conn *Conn)

func WithProxy(proxy func(req *http.Request) (*url.URL, error)) Option {
	return func(conn *Conn) {
		conn.proxy = proxy
	}
}
func WithWriteTimeout(timeout time.Duration) Option {
	return func(conn *Conn) {
		conn.writeTimeout = timeout
	}
}
func WithMsgType(mt MsgType) Option {
	return func(conn *Conn) {
		conn.mt = mt
	}
}
