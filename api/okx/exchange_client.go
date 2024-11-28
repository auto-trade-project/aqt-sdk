package okx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/kurosann/aqt-sdk/api"
)

type ExchangeClient struct {
	pc  *PublicClient
	bc  *BusinessClient
	pvc *PrivateClient
	rc  *PublicRestClient

	wsUrls      map[Destination]map[SvcType]BaseURL
	restUrls    map[Destination]BaseURL
	env         Destination
	keyConfig   OkxKeyConfig
	proxy       func(req *http.Request) (*url.URL, error)
	log         api.ILogger
	readMonitor func(arg Arg)

	// 是否为带单交易
	isCopyTrading bool
}

func (w *ExchangeClient) GetMarketName() string {
	return "okex"
}

func (w *ExchangeClient) PlaceOrder(ctx context.Context, req api.PlaceOrderReq) (*api.PlaceOrder, error) {
	order, err := w.rc.PlaceOrder(ctx, PlaceOrderReq{
		InstID:  req.TokenType,
		ClOrdID: req.InternalOrderId,
		Sz:      req.Sz,
		Px:      req.Px,
		Side:    req.Side,
		TdMode:  "cash",
		OrdType: req.OrdType,
	})
	if err != nil {
		msgs := make([]string, 0)
		for _, item := range order.Data {
			if item.SMsg != "" {
				msgs = append(msgs, item.SMsg)
			}
		}
		return nil, w.genErrMsg("code:%s %w: %s", order.Code, err, strings.Join(msgs, "; "))
	}
	if len(order.Data) == 0 {
		return nil, w.genErrMsg("place order failed")
	}
	return &api.PlaceOrder{
		OrderId: order.Data[0].OrdId,
	}, nil
}

func (w *ExchangeClient) QueryOrder(ctx context.Context, req api.GetOrderReq) (*api.Order, error) {
	order, err := w.rc.GetOrder(ctx, PlaceOrderReq{
		InstID:  req.TokenType,
		ClOrdID: req.OrderId,
	})
	if err != nil {
		return nil, err
	}
	if len(order.Data) == 0 {
		return nil, w.genErrMsg("order not found")
	}
	timestampString := order.Data[0].CTime
	// 将时间戳字符串转换为整数
	timestampInt, _ := strconv.ParseInt(timestampString, 10, 64)
	return &api.Order{
		TokenType:       order.Data[0].InstId,
		ExOrderId:       order.Data[0].OrdId,
		InternalOrderId: order.Data[0].ClOrdId,
		Side:            order.Data[0].Side,
		Fee:             order.Data[0].Fee,
		Px:              order.Data[0].Px,
		Sz:              order.Data[0].Sz,
		State:           stateToOrderState(order.Data[0].State),
		Time:            time.UnixMilli(timestampInt),
	}, nil
}

func (w *ExchangeClient) CancelOrder(ctx context.Context, tokenType, orderId string) error {
	_, err := w.rc.CancelOrder(ctx, tokenType, orderId)
	if err != nil {
		return w.genErrMsg("cancel order failed: %w", err)
	}
	return nil
}

func (w *ExchangeClient) QueryCandles(ctx context.Context, req api.CandlesReq) ([]*api.Candle, error) {
	candles, err := w.rc.Candles(ctx, CandlesticksReq{
		InstID: req.TokenType,
		Before: req.StartTime.UnixMilli(),
		After:  req.EndTime.UnixMilli(),
		Limit:  req.Limit,
		Bar:    w.TimeFrameToBar(req.TimeFrame),
	})
	if err != nil {
		return nil, w.genErrMsg("query candles failed: %w", err)
	}
	if len(candles.Data) == 0 {
		return nil, w.genErrMsg("candles not found")
	}
	res := make([]*api.Candle, len(candles.Data))
	for i, datum := range candles.Data {
		timestampInt, _ := strconv.ParseInt(datum.Ts, 10, 64)
		res[i] = &api.Candle{
			TokenType:   req.TokenType,
			Ts:          time.UnixMilli(timestampInt),
			O:           datum.O,
			H:           datum.H,
			L:           datum.L,
			C:           datum.C,
			Vol:         datum.Vol,
			VolCcy:      datum.VolCcy,
			VolCcyQuote: datum.VolCcyQuote,
			Confirm:     datum.Confirm,
		}
	}
	return res, nil
}

func (w *ExchangeClient) AssetListen(ctx context.Context, callback func(resp *api.Asset)) error {
	return w.pvc.Account(ctx, func(resp *WsResp[*Balance]) {
		for _, balance := range resp.Data {
			for _, datum := range balance.Details {
				callback(&api.Asset{
					TokenType: datum.Ccy,
					Balance:   datum.CashBal,
					AvailBal:  datum.AvailBal,
					FrozenBal: datum.FrozenBal,
				})
			}
		}
	})
}

const (
	Bar1m  = "1m"
	Bar15m = "15m"
	Bar1H  = "1H"
	Bar2H  = "2H"
	Bar4H  = "4H"
	Bar1D  = "1D"

	TimeFrame1m  = time.Minute
	TimeFrame15m = time.Minute * 15
	TimeFrame1H  = time.Hour
	TimeFrame2H  = time.Hour * 2
	TimeFrame4H  = time.Hour * 4
	TimeFrame1D  = time.Hour * 24
)

// TimeFrameToBar 将时间周期转换为k线图时间周期
func (w *ExchangeClient) TimeFrameToBar(timeFrame time.Duration) string {
	bar := Bar1H
	switch {
	case timeFrame <= TimeFrame1m:
		bar = Bar1m
	case timeFrame <= TimeFrame15m:
		bar = Bar15m
	case timeFrame <= TimeFrame1H:
		bar = Bar1H
	case timeFrame <= TimeFrame2H:
		bar = Bar2H
	case timeFrame <= TimeFrame4H:
		bar = Bar4H
	case timeFrame <= TimeFrame1D:
		bar = Bar1D
	}
	return bar
}

func (w *ExchangeClient) CandleListen(ctx context.Context, timeFrame time.Duration, tokenType string, callback func(resp *api.Candle)) error {
	err := w.bc.Candle(ctx, w.TimeFrameToBar(timeFrame), tokenType, func(resp *WsResp[*Candle]) {
		for _, datum := range resp.Data {
			timestampInt, _ := strconv.ParseInt(datum.Ts, 10, 64)
			callback(&api.Candle{
				TokenType: tokenType,
				Ts:        time.UnixMilli(timestampInt),
				O:         datum.O,
				H:         datum.H,
				L:         datum.L,
				C:         datum.C,
				Vol:       datum.Vol,
				VolCcy:    datum.VolCcy,
			})
		}
	})
	if err != nil {
		return w.genErrMsg("candle listen failed: %w", err)
	}
	return nil
}

func (w *ExchangeClient) MarkPriceListen(ctx context.Context, tokenType string, callback func(resp *api.MarkPrice)) error {
	err := w.pc.MarkPrice(ctx, tokenType, func(resp *WsResp[*MarkPrice]) {
		for _, datum := range resp.Data {
			t, _ := strconv.Atoi(datum.Ts)
			callback(&api.MarkPrice{
				Px:        datum.MarkPx,
				TokenType: tokenType,
				Ts:        time.UnixMilli(int64(t)),
			})
		}
	})
	if err != nil {
		return w.genErrMsg("mark price listen failed: %w", err)
	}
	return nil
}

func (w *ExchangeClient) OrderListen(ctx context.Context, callback func(resp *api.Order)) error {
	err := w.pvc.SpotOrders(ctx, func(resp *WsResp[*Order]) {
		for _, datum := range resp.Data {
			timestampString := datum.FillTime
			// 将时间戳字符串转换为整数
			timestampInt, _ := strconv.ParseInt(timestampString, 10, 64)
			callback(&api.Order{
				Px:              datum.Px,
				TokenType:       datum.InstId,
				ExOrderId:       datum.OrdId,
				InternalOrderId: datum.ClOrdId,
				Side:            datum.Side,
				Fee:             datum.Fee,
				Sz:              datum.Sz,
				State:           stateToOrderState(datum.State),
				Time:            time.UnixMilli(timestampInt),
			})
		}
	})
	if err != nil {
		return w.genErrMsg("order listen failed: %w", err)
	}
	return nil
}
func stateToOrderState(state string) api.OrderState {
	switch state {
	case "canceled", "mmp_canceled":
		return api.OrderStateCanceled
	case "live":
		return api.OrderStateOpen
	case "partially_filled":
		return api.OrderStatePartiallyFilled
	case "filled":
		return api.OrderStateFilled
	}
	return api.OrderStateOpen
}

func (w *ExchangeClient) ReadMonitor(readMonitor func(arg string)) {
	f := func(arg Arg) {
		valueOfArg := reflect.ValueOf(arg)
		strs := make([]string, 0, valueOfArg.NumField())
		for i := 0; i < valueOfArg.NumField(); i++ {
			field := valueOfArg.Field(i)
			if field.String() != "" {
				strs = append(strs, field.String())
			}
		}
		readMonitor(strings.Join(strs, ":"))
	}
	w.readMonitor = f
}

func (w *ExchangeClient) SetLog(log api.ILogger) {
	w.log = log
}

func (w *ExchangeClient) genErrMsg(format string, args ...any) error {
	return fmt.Errorf("[%s] "+format, append([]any{w.GetMarketName()}, args...)...)
}

func NewExchangeClient(ctx context.Context, opts ...api.Opt) (*ExchangeClient, error) {
	client := ExchangeClient{
		proxy:       http.ProxyFromEnvironment,
		wsUrls:      DefaultWsUrls,
		restUrls:    DefaultRestUrl,
		env:         NormalServer,
		log:         api.DefaultLogger{},
		readMonitor: func(arg Arg) {},
	}
	for _, opt := range opts {
		opt(&client)
	}
	if client.keyConfig.Apikey == "" {
		return nil, errors.New("not config: Apikey or Secretkey or Passphrase")
	}
	client.pc = &PublicClient{BaseWsClient: *NewBaseWsClient(ctx, WsInfo{
		typ:         Public,
		url:         client.wsUrls[client.env][Public],
		keyConfig:   client.keyConfig,
		proxy:       client.proxy,
		log:         client.log,
		readMonitor: client.readMonitor,
	})}
	client.bc = &BusinessClient{BaseWsClient: *NewBaseWsClient(ctx, WsInfo{
		typ:         Business,
		url:         client.wsUrls[client.env][Business],
		keyConfig:   client.keyConfig,
		proxy:       client.proxy,
		log:         client.log,
		readMonitor: client.readMonitor,
	})}
	client.pvc = &PrivateClient{BaseWsClient: *NewBaseWsClient(ctx, WsInfo{
		typ:         Private,
		url:         client.wsUrls[client.env][Private],
		keyConfig:   client.keyConfig,
		proxy:       client.proxy,
		log:         client.log,
		readMonitor: client.readMonitor,
	})}
	client.rc = &PublicRestClient{RestClient: NewRestClient(ctx, client.keyConfig, client.env, client.restUrls, client.proxy)}
	return &client, nil
}
