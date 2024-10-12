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
func WithKeepalive(timeout time.Duration, keepalivePingFn func(*Conn) error, keepalivePongFn func([]byte) bool) Option {
	return func(conn *Conn) {
		conn.keepaliveTimeout = timeout
		conn.keepalivePingFn = keepalivePingFn
		conn.keepalivePongFn = keepalivePongFn
	}
}
func WithMsgType(mt MsgType) Option {
	return func(conn *Conn) {
		conn.mt = mt
	}
}
