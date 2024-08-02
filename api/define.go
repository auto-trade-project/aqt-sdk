package api

import (
	"context"
	"github.com/kurosann/aqt-sdk/api/okx"
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

type IMarketApi interface {
	IMarketClient
	IMarketExClient
}
type IMarketExClient interface {
	SetLog(logger ILogger)
	ReadMonitor(f func(arg okx.Arg))
	Account(ctx context.Context, callback func(resp Balance)) error
	Candle(ctx context.Context, channel, instId string, callback func(resp *Candle)) error
	MarkPrice(ctx context.Context, instId string, callback func(resp MarkPrice)) error
	SpotOrders(ctx context.Context, callback func(resp Order)) error
}

type IMarketClient interface {
	PlaceOrder(ctx context.Context, req PlaceOrderReq) (PlaceOrder, error)
	GetOrder(ctx context.Context, req GetOrderReq) (Order, error)
	CancelOrder(ctx context.Context, tokenType, orderId string) error
	Candles(ctx context.Context, req CandlesReq) ([]*Candle, error)
}

type MarkPrice struct {
	TokenType string
	Px        string
	Ts        string
}

type Candle struct {
	TokenType   string
	Ts          string
	O           string
	H           string
	L           string
	C           string
	Vol         string
	VolCcy      string
	VolCcyQuote string
	Confirm     string
}
type Balance struct {
	TokenType string
	Balance   string
	AvailBal  string
	FrozenBal string
}

type Order struct {
	TokenType  string
	PlmOrderId string // 平台订单id
	SysOrdorId string // 系统订单id
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
type Opt func(api IMarketApi)

type OptInfo struct {
	Exchange
	Opts []Opt
}
