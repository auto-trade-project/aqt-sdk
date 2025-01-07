package api

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type (
	Exchange string
)

const (
	OkxExchange = Exchange("okx")
	//BalanceExchange = Exchange("balance")
)

type IMarketClient interface {
	IMarketUnaryClient
	IMarketStreamClient
	GetMarketName() string
}

type IMarketStreamClient interface {
	AssetListen(ctx context.Context, callback func(resp *Asset)) error
	CandleListen(ctx context.Context, timeFrame time.Duration, tokenType string, callback func(resp *Candle)) error
	MarkPriceListen(ctx context.Context, tokenType string, callback func(resp *MarkPrice)) error
	OrderListen(ctx context.Context, callback func(resp *Order)) error
}
type IMarketUnaryClient interface {
	PlaceOrder(ctx context.Context, req PlaceOrderReq) (*PlaceOrder, error)
	QueryOrder(ctx context.Context, req GetOrderReq) (*Order, error)
	CancelOrder(ctx context.Context, tokenType, orderId string) error
	QueryCandles(ctx context.Context, req CandlesReq) ([]*Candle, error)
}

type ISetLogger interface {
	SetLog(logger ILogger)
}
type IReadMonitor interface {
	ReadMonitor(f func(arg string))
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

var (
	OrderNotFound = errors.New("order not found")
)

type ILogger interface {
	Infof(template string, args ...interface{})
	Debugf(template string, args ...interface{})
	Warnf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
	Panicf(template string, args ...interface{})
}
type DefaultLogger struct{}

func (l DefaultLogger) Infof(template string, args ...interface{}) {
	fmt.Printf(template, args...)
}
func (l DefaultLogger) Debugf(template string, args ...interface{}) {
	fmt.Printf(template, args...)
}
func (l DefaultLogger) Warnf(template string, args ...interface{}) {
	fmt.Printf(template, args...)
}
func (l DefaultLogger) Errorf(template string, args ...interface{}) {
	fmt.Printf(template, args...)
}
func (l DefaultLogger) Panicf(template string, args ...interface{}) {
	fmt.Printf(template, args...)
}

type MarkPrice struct {
	TokenType string
	Px        string
	Ts        time.Time
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

type OrderState string

const (
	OrderStateOpen            OrderState = "open"             // 未成交
	OrderStateCanceled        OrderState = "canceled"         // 撤单成功
	OrderStatePartiallyFilled OrderState = "partially_filled" // 部分成交
	OrderStateFilled          OrderState = "filled"           // 完全成交
)

type Order struct {
	TokenType       string
	ExOrderId       string // 交易所订单id
	InternalOrderId string // 程序内部订单id
	Side            string
	Fee             string
	Px              string
	Sz              string
	State           OrderState
	Time            time.Time
}
type PlaceOrderReq struct {
	TokenType       string
	InternalOrderId string
	Sz              string
	Px              string
	Side            string
	OrdType         string
	TdMode    string
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
	TimeFrame time.Duration
}
type Opt func(api IMarketClient)

type OptInfo struct {
	Exchange
	Opts []Opt
}

func NewOptInfo(exchange Exchange, opts ...Opt) OptInfo {
	return OptInfo{Exchange: exchange, Opts: opts}
}
