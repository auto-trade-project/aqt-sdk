package okx

import (
	"context"
)

type BusinessClient struct {
	BaseWsClient
}

func (w *BusinessClient) MarkPriceCandlesticks(ctx context.Context, channel, instId string, callback func(resp *WsResp[*MarkPriceCandle])) error {
	if err := w.Login(ctx); err != nil {
		return err
	}
	return Subscribe(&w.BaseWsClient, ctx, MakeArg("candle"+channel, instId), callback)
}
func (w *BusinessClient) UMarkPriceCandlesticks(channel, instId string) error {
	return w.Unsubscribe(MakeArg("mark-price-candle"+channel, instId))
}
func (w *BusinessClient) Candle(ctx context.Context, channel, instId string, callback func(resp *WsResp[*Candle])) error {
	return Subscribe(&w.BaseWsClient, ctx, MakeArg("candle"+channel, instId), callback)
}
func (w *BusinessClient) UCandle(channel, instId string) error {
	return w.Unsubscribe(MakeArg("candle"+channel, instId))
}

type PublicClient struct {
	BaseWsClient
}

func (w *PublicClient) MarkPrice(ctx context.Context, instId string, callback func(resp *WsResp[*MarkPrice])) error {
	return Subscribe(&w.BaseWsClient, ctx, MakeArg("mark-price", instId), callback)
}
func (w *PublicClient) UMarkPrice(instId string) error {
	return w.Unsubscribe(MakeArg("mark-price", instId))
}

func (w *BusinessClient) OrderBook(ctx context.Context, channel, sprdId string, callback func(resp *WsResp[*OrderBook])) error {
	return Subscribe(&w.BaseWsClient, ctx, MakeSprdArg(channel, sprdId), callback)
}
func (w *BusinessClient) UOrderBook(channel, instId string) error {
	return w.Unsubscribe(MakeSprdArg(channel, instId))
}

//-------------------------- 交易 --------------------------

type PrivateClient struct {
	BaseWsClient
}

// Account 资金频道
func (w *PrivateClient) Account(ctx context.Context, callback func(resp *WsResp[*Balance])) error {
	if err := w.Login(ctx); err != nil {
		return err
	}
	return Subscribe(&w.BaseWsClient, ctx, MakeArg("account", ""), callback)
}
func (w *PrivateClient) UAccount() error {
	return w.Unsubscribe(MakeArg("account", ""))
}

// Positions 持仓频道
func (w *PrivateClient) Positions(ctx context.Context, callback func(resp *WsResp[*Balances])) error {
	if err := w.Login(ctx); err != nil {
		return err
	}
	return Subscribe(&w.BaseWsClient, ctx, MakeArg("positions", ""), callback)
}

// Trades 成交订单频道
func (w *BusinessClient) Trades(ctx context.Context, sprdId string, callback func(resp *WsResp[*Trades])) error {
	if err := w.Login(ctx); err != nil {
		return err
	}
	return Subscribe(&w.BaseWsClient, ctx, MakeSprdArg("sprd-trades", sprdId), callback)
}

// UTrades 成交订单频道
func (w *BusinessClient) UTrades(sprdId string) error {
	return w.Unsubscribe(MakeSprdArg("sprd-trades", sprdId))
}

// Orders 撮合交易订单频道
func (w *PrivateClient) Orders(ctx context.Context, instType string, callback func(resp *WsResp[*Order])) error {
	if err := w.Login(ctx); err != nil {
		return err
	}
	return Subscribe(&w.BaseWsClient, ctx, MakeArg("orders", instType), callback)
}

// UOrders 取消订阅撮合交易订单频道
func (w *PrivateClient) UOrders(instType string) error {
	return w.Unsubscribe(MakeArg("orders", instType))
}

// SpotOrders 撮合交易订单频道
func (w *PrivateClient) SpotOrders(ctx context.Context, callback func(resp *WsResp[*Order])) error {
	if err := w.Login(ctx); err != nil {
		return err
	}
	return Subscribe(&w.BaseWsClient, ctx, &Arg{
		Channel:  "orders",
		InstType: "SPOT",
	}, callback)
}

// USpotOrders 取消订阅撮合交易订单频道
func (w *PrivateClient) USpotOrders() error {
	return w.Unsubscribe(MakeArg("orders", "SPOT"))
}
