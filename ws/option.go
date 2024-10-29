package ws

import (
	"net/http"
	"net/url"
	"time"
)

type Option func(conn *Conn)

func WithProxy(proxy func(req *http.Request) (*url.URL, error)) Option {
	return func(conn *Conn) {
		conn.config.proxy = proxy
	}
}
func WithWriteTimeout(timeout time.Duration) Option {
	return func(conn *Conn) {
		conn.config.writeTimeout = timeout
	}
}
func WithKeepalive(timeout time.Duration, keepalivePingFn func(*Conn) error, keepalivePongFn func([]byte) bool) Option {
	return func(conn *Conn) {
		conn.config.keepaliveTimeout = timeout
		conn.config.keepalivePingFn = keepalivePingFn
		conn.config.keepalivePongFn = keepalivePongFn
	}
}
func WithMsgType(msgType MsgType) Option {
	return func(conn *Conn) {
		conn.config.msgType = msgType
	}
}
