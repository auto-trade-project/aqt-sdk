package okx

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/kurosann/aqt-sdk/api"
)

type ExchangeClient struct {
	pc        *PublicClient
	bc        *BusinessClient
	pvc       *PrivateClient
	rc        *PublicRestClient
	wsUrls    map[Destination]map[SvcType]BaseURL
	restUrls  map[Destination]BaseURL
	env       Destination
	keyConfig OkxKeyConfig
	proxy     func(req *http.Request) (*url.URL, error)
}

func (w ExchangeClient) GetMarketName() string {
	return "okex"
}

func (w ExchangeClient) PlaceOrder(ctx context.Context, req api.PlaceOrderReq) (*api.PlaceOrder, error) {
	order, err := w.rc.PlaceOrder(ctx, PlaceOrderReq{
		InstID:  req.TokenType,
		ClOrdID: req.ClOrdID,
		Sz:      req.Sz,
		Px:      req.Px,
		Side:    req.Side,
		TdMode:  "cash",
		OrdType: req.OrdType,
	})
	if err != nil {
		return nil, err
	}
	if len(order.Data) == 0 {
		return nil, nil
	}
	return &api.PlaceOrder{
		OrderId: order.Data[0].OrdId,
	}, nil
}

func (w ExchangeClient) QueryOrder(ctx context.Context, req api.GetOrderReq) (*api.Order, error) {
	order, err := w.rc.GetOrder(ctx, PlaceOrderReq{
		InstID:  req.TokenType,
		ClOrdID: req.OrderId,
	})
	if err != nil {
		return nil, err
	}
	if len(order.Data) == 0 {
		return nil, nil
	}
	timestampString := order.Data[0].CTime
	// 将时间戳字符串转换为整数
	timestampInt, _ := strconv.ParseInt(timestampString, 10, 64)
	return &api.Order{
		TokenType:  order.Data[0].InstId,
		PlmOrderId: order.Data[0].OrdId,
		SysOrderId: order.Data[0].ClOrdId,
		Side:       order.Data[0].Side,
		Fee:        order.Data[0].Fee,
		Px:         order.Data[0].Px,
		Sz:         order.Data[0].Sz,
		State:      order.Data[0].State,
		Time:       time.UnixMilli(timestampInt),
	}, nil
}

func (w ExchangeClient) CancelOrder(ctx context.Context, tokenType, orderId string) error {
	_, err := w.rc.CancelOrder(ctx, tokenType, orderId)
	if err != nil {
		return err
	}
	return nil
}

func (w ExchangeClient) QueryCandles(ctx context.Context, req api.CandlesReq) ([]*api.Candle, error) {
	candles, err := w.rc.Candles(ctx, CandlesticksReq{
		InstID: req.TokenType,
		Before: req.StartTime.UnixMilli(),
		After:  req.EndTime.UnixMilli(),
		Limit:  req.Limit,
		Bar:    req.Norm,
	})
	if err != nil {
		return nil, err
	}
	if len(candles.Data) == 0 {
		return nil, nil
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

func (w ExchangeClient) AssetListen(ctx context.Context, callback func(resp *api.Asset)) error {
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

func (w ExchangeClient) CandleListen(ctx context.Context, channel, instId string, callback func(resp *api.Candle)) error {
	return w.bc.Candle(ctx, channel, instId, func(resp *WsResp[*Candle]) {
		for _, datum := range resp.Data {
			timestampInt, _ := strconv.ParseInt(datum.Ts, 10, 64)
			callback(&api.Candle{
				TokenType: instId,
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
}

func (w ExchangeClient) MarkPriceListen(ctx context.Context, instId string, callback func(resp *api.MarkPrice)) error {
	return w.pc.MarkPrice(ctx, instId, func(resp *WsResp[*MarkPrice]) {
		for _, datum := range resp.Data {
			callback(&api.MarkPrice{
				Px:        datum.MarkPx,
				TokenType: instId,
				Ts:        datum.Ts,
			})
		}
	})
}

func (w ExchangeClient) OrderListen(ctx context.Context, callback func(resp *api.Order)) error {
	return w.pvc.SpotOrders(ctx, func(resp *WsResp[*Order]) {
		for _, datum := range resp.Data {
			timestampString := datum.FillTime
			// 将时间戳字符串转换为整数
			timestampInt, _ := strconv.ParseInt(timestampString, 10, 64)
			callback(&api.Order{
				Px:         datum.Px,
				TokenType:  datum.InstId,
				PlmOrderId: datum.OrdId,
				SysOrderId: datum.ClOrdId,
				Side:       datum.Side,
				Fee:        datum.Fee,
				Sz:         datum.Sz,
				State:      datum.State,
				Time:       time.UnixMilli(timestampInt),
			})
		}
	})
}

func (w ExchangeClient) ReadMonitor(readMonitor func(arg string)) {
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
	w.pc.ReadMonitor = f
	w.bc.ReadMonitor = f
	w.pvc.ReadMonitor = f
}

func (w ExchangeClient) SetLog(log api.ILogger) {
	w.pc.Log = log
	w.bc.Log = log
	w.pvc.Log = log
}

func NewExchangeClient(ctx context.Context, opts ...api.Opt) (*ExchangeClient, error) {
	client := ExchangeClient{
		proxy:    http.ProxyFromEnvironment,
		wsUrls:   DefaultWsUrls,
		restUrls: DefaultRestUrl,
		env:      NormalServer,
	}
	for _, opt := range opts {
		opt(&client)
	}
	if client.keyConfig.Apikey == "" {
		return nil, errors.New("not config: Apikey or Secretkey or Passphrase")
	}
	client.pc = &PublicClient{BaseWsClient: NewBaseWsClient(ctx, Public, client.wsUrls[client.env][Public], client.keyConfig, client.proxy)}
	client.bc = &BusinessClient{BaseWsClient: NewBaseWsClient(ctx, Business, client.wsUrls[client.env][Business], client.keyConfig, client.proxy)}
	client.pvc = &PrivateClient{BaseWsClient: NewBaseWsClient(ctx, Private, client.wsUrls[client.env][Private], client.keyConfig, client.proxy)}
	client.rc = &PublicRestClient{RestClient: NewRestClient(ctx, client.keyConfig, client.env, client.restUrls, client.proxy)}
	return &client, nil
}
