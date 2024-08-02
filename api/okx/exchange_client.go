package okx

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/kurosann/aqt-sdk/api"
)

type ExchangeClient struct {
	pc        *PublicClient
	bc        *BusinessClient
	pvc       *PrivateClient
	rc        *PublicRestClient
	urls      map[Destination]map[SvcType]BaseURL
	env       Destination
	keyConfig OkxKeyConfig
	proxy     func(req *http.Request) (*url.URL, error)
}

func (w *ExchangeClient) PlaceOrder(ctx context.Context, req api.PlaceOrderReq) (api.PlaceOrder, error) {
	order, err := w.rc.PlaceOrder(ctx, PlaceOrderReq{
		InstID:  req.TokenType,
		ClOrdID: req.ClOrdID,
		Sz:      req.Sz,
		Px:      req.Px,
		Side:    req.Side,
		OrdType: req.OrdType,
	})
	if err != nil {
		return api.PlaceOrder{}, err
	}
	if len(order.Data) == 0 {
		return api.PlaceOrder{}, nil
	}
	return api.PlaceOrder{
		OrderId: order.Data[0].OrdId,
	}, nil
}

func (w *ExchangeClient) GetOrder(ctx context.Context, req api.GetOrderReq) (api.Order, error) {
	//TODO implement me
	order, err := w.rc.GetOrder(ctx, PlaceOrderReq{
		InstID:  req.TokenType,
		ClOrdID: req.OrderId,
	})
	if err != nil {
		return api.Order{}, err
	}
	if len(order.Data) == 0 {
		return api.Order{}, nil
	}
	timestampString := order.Data[0].CTime
	// 将时间戳字符串转换为整数
	timestampInt, _ := strconv.ParseInt(timestampString, 10, 64)
	return api.Order{
		TokenType:  order.Data[0].InstId,
		PlmOrderId: order.Data[0].OrdId,
		SysOrdorId: order.Data[0].ClOrdId,
		Side:       order.Data[0].Side,
		Fee:        order.Data[0].Fee,
		Px:         order.Data[0].Px,
		Sz:         order.Data[0].Sz,
		State:      order.Data[0].State,
		Time:       time.UnixMilli(timestampInt),
	}, nil
}

func (w *ExchangeClient) CancelOrder(ctx context.Context, tokenType, orderId string) error {
	_, err := w.rc.CancelOrder(ctx, tokenType, orderId)
	if err != nil {
		return err
	}
	return nil
}

func (w *ExchangeClient) Candles(ctx context.Context, req api.CandlesReq) ([]*api.Candle, error) {
	//TODO implement me
	candles, err := w.rc.Candles(ctx, CandlesticksReq{
		InstID: req.TokenType,
		Before: req.StartTime.UnixMilli(),
		After:  req.EndTime.UnixMilli(),
		Limit:  req.Limit,
	})
	if err != nil {
		return nil, err
	}
	if len(candles.Data) == 0 {
		return nil, nil
	}
	res := make([]*api.Candle, len(candles.Data))
	for i, datum := range candles.Data {
		res[i] = &api.Candle{
			TokenType:   req.TokenType,
			Ts:          datum.Ts,
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

func (w *ExchangeClient) Account(ctx context.Context, callback func(resp api.Balance)) error {
	//TODO implement me
	return w.pvc.Account(ctx, func(resp *WsResp[*Balance]) {
		for _, balance := range resp.Data {
			for _, datum := range balance.Details {
				callback(api.Balance{
					TokenType: datum.Ccy,
					Balance:   datum.CashBal,
					AvailBal:  datum.AvailBal,
					FrozenBal: datum.FrozenBal,
				})
			}
		}
	})
}

func (w *ExchangeClient) Candle(ctx context.Context, channel, instId string, callback func(resp *api.Candle)) error {
	return w.bc.Candle(ctx, channel, instId, func(resp *WsResp[*Candle]) {
		for _, datum := range resp.Data {
			callback(&api.Candle{
				TokenType: instId,
				Ts:        datum.Ts,
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

func (w *ExchangeClient) MarkPrice(ctx context.Context, instId string, callback func(resp api.MarkPrice)) error {
	//TODO implement me
	return w.pc.MarkPrice(ctx, instId, func(resp *WsResp[*MarkPrice]) {
		for _, datum := range resp.Data {
			callback(api.MarkPrice{
				Px:        datum.MarkPx,
				TokenType: instId,
				Ts:        datum.Ts,
			})
		}
	})
}

func (w *ExchangeClient) SpotOrders(ctx context.Context, callback func(resp api.Order)) error {
	//TODO implement me
	return w.pvc.SpotOrders(ctx, func(resp *WsResp[*Order]) {
		for _, datum := range resp.Data {
			timestampString := datum.FillTime
			// 将时间戳字符串转换为整数
			timestampInt, _ := strconv.ParseInt(timestampString, 10, 64)
			callback(api.Order{
				Px:         datum.Px,
				TokenType:  datum.InstId,
				PlmOrderId: datum.OrdId,
				SysOrdorId: datum.ClOrdId,
				Side:       datum.Side,
				Fee:        datum.Fee,
				Sz:         datum.Sz,
				State:      datum.State,
				Time:       time.UnixMilli(timestampInt),
			})
		}
	})
}

func (w *ExchangeClient) ReadMonitor(readMonitor func(arg Arg)) {
	w.pc.ReadMonitor = readMonitor
	w.bc.ReadMonitor = readMonitor
	w.pvc.ReadMonitor = readMonitor
}

func (w *ExchangeClient) SetLog(log api.ILogger) {
	w.pc.Log = log
	w.bc.Log = log
	w.pvc.Log = log
}

func (w *ExchangeClient) Close() {

}

func NewExchangeClient(ctx context.Context, opts ...api.Opt) (*ExchangeClient, error) {
	client := ExchangeClient{
		proxy: http.ProxyFromEnvironment,
		urls:  DefaultWsUrls,
		env:   NormalServer,
	}
	for _, opt := range opts {
		opt(&client)
	}
	if client.keyConfig.Apikey == "" {
		return nil, errors.New("not config: Apikey or Secretkey or Passphrase")
	}
	client.pc = &PublicClient{BaseWsClient: NewBaseWsClient(ctx, Public, client.urls[client.env][Public], client.keyConfig, client.proxy)}
	client.bc = &BusinessClient{BaseWsClient: NewBaseWsClient(ctx, Business, client.urls[client.env][Business], client.keyConfig, client.proxy)}
	client.pvc = &PrivateClient{BaseWsClient: NewBaseWsClient(ctx, Private, client.urls[client.env][Private], client.keyConfig, client.proxy)}
	client.rc = &PublicRestClient{RestClient: NewRestClient(ctx, client.keyConfig, client.env, client.proxy)}
	return &client, nil
}
