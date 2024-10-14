package api

import (
	"context"
	"time"
)

type (
	Exchange string
)

const (
	OkxExchange = Exchange("okx")
	//BalanceExchange = Exchange("balance")
)

type ILogger interface {
	Infof(template string, args ...interface{})
	Debugf(template string, args ...interface{})
	Warnf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
	Panicf(template string, args ...interface{})
}

type IMarketClient interface {
	IMarketUnaryClient
	IMarketStreamClient
	GetMarketName() string
}
type IMarketStreamClient interface {
	SetLog(logger ILogger)
	ReadMonitor(f func(arg string))
	AssetListen(ctx context.Context, callback func(resp *Asset)) error
	CandleListen(ctx context.Context, channel, tokenType string, callback func(resp *Candle)) error
	MarkPriceListen(ctx context.Context, instId string, callback func(resp *MarkPrice)) error
	OrderListen(ctx context.Context, callback func(resp *Order)) error
}
type IMarketUnaryClient interface {
	PlaceOrder(ctx context.Context, req PlaceOrderReq) (*PlaceOrder, error)
	QueryOrder(ctx context.Context, req GetOrderReq) (*Order, error)
	CancelOrder(ctx context.Context, tokenType, orderId string) error
	QueryCandles(ctx context.Context, req CandlesReq) ([]*Candle, error)
}

type NewStreamClient interface {
	New(conn IConnect) IMarketStreamClient
}
type Dial interface {
	Dial() (IConnect, error)
}
type IConnect interface {
	Open() error
	IsAlive() error
	Reload() error
}

type MarkPrice struct {
	TokenType string
	Px        string
	Ts        string
}

type Candle struct {
	TokenType   string
	Ts          time.Time
	O           string
	H           string
	L           string
	C           string
	Vol         string
	VolCcy      string
	VolCcyQuote string
	Confirm     string
}
type Asset struct {
	TokenType string
	Balance   string
	AvailBal  string
	FrozenBal string
}

type Order struct {
	TokenType  string
	PlmOrderId string // 平台订单id
	SysOrderId string // 系统订单id
	Side       string
	Fee        string
	Px         string
	Sz         string
	State      string
	Time       time.Time
}
type PlaceOrderReq struct {
	TokenType string
	ClOrdID   string
	Sz        string
	Px        string
	Side      string
	OrdType   string
}
type GetOrderReq struct {
	TokenType string
	OrderId   string
}
type PlaceOrder struct {
	OrderId string
}
type CandlesReq struct {
	TokenType string
	EndTime   time.Time
	StartTime time.Time
	Limit     int64
	Norm      string
}
type Opt func(api IMarketClient)

type OptInfo struct {
	Exchange
	Opts []Opt
}

func NewOptInfo(exchange Exchange, opts ...Opt) OptInfo {
	return OptInfo{Exchange: exchange, Opts: opts}
}
