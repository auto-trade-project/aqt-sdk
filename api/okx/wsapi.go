package okx

import "context"

type BusinessClient struct {
	WsClient
}

func (w *BusinessClient) MarkPriceCandlesticks(ctx context.Context, channel, instId string, callback func(resp *WsResp[*MarkPriceCandle])) error {
	if err := w.login(ctx); err != nil {
		return err
	}
	return Subscribe(&w.WsClient, ctx, makeArg("candle"+channel, instId), callback)
}
func (w *BusinessClient) UMarkPriceCandlesticks(channel, instId string) error {
	return w.unsubscribe(makeArg("mark-price-candle"+channel, instId))
}
func (w *BusinessClient) Candle(ctx context.Context, channel, instId string, callback func(resp *WsResp[*Candle])) error {
	return Subscribe(&w.WsClient, ctx, makeArg("candle"+channel, instId), callback)
}
func (w *BusinessClient) UCandle(channel, instId string) error {
	return w.unsubscribe(makeArg("candle"+channel, instId))
}

type PublicClient struct {
	WsClient
}

func (w *PublicClient) MarkPrice(ctx context.Context, instId string, callback func(resp *WsResp[*MarkPrice])) error {
	return Subscribe(&w.WsClient, ctx, makeArg("mark-price", instId), callback)
}
func (w *PublicClient) UMarkPrice(instId string) error {
	return w.unsubscribe(makeArg("mark-price", instId))
}

func (w *BusinessClient) OrderBook(ctx context.Context, channel, sprdId string, callback func(resp *WsResp[*OrderBook])) error {
	return Subscribe(&w.WsClient, ctx, makeSprdArg(channel, sprdId), callback)
}
func (w *BusinessClient) UOrderBook(channel, instId string) error {
	return w.unsubscribe(makeSprdArg(channel, instId))
}

//-------------------------- 交易 --------------------------

type PrivateClient struct {
	WsClient
}

// Account 资金频道
func (w *PrivateClient) Account(ctx context.Context, callback func(resp *WsResp[*Balance])) error {
	if err := w.login(ctx); err != nil {
		return err
	}
	return Subscribe(&w.WsClient, ctx, makeArg("account", ""), callback)
}
func (w *PrivateClient) UAccount() error {
	return w.unsubscribe(makeArg("account", ""))
}

// Positions 持仓频道
func (w *PrivateClient) Positions(ctx context.Context, callback func(resp *WsResp[*Balances])) error {
	if err := w.login(ctx); err != nil {
		return err
	}
	return Subscribe(&w.WsClient, ctx, makeArg("positions", ""), callback)
}

// Trades 成交订单频道
func (w *BusinessClient) Trades(ctx context.Context, sprdId string, callback func(resp *WsResp[*Trades])) error {
	if err := w.login(ctx); err != nil {
		return err
	}
	return Subscribe(&w.WsClient, ctx, makeSprdArg("sprd-trades", sprdId), callback)
}

// UTrades 成交订单频道
func (w *BusinessClient) UTrades(sprdId string) error {
	return w.unsubscribe(makeSprdArg("sprd-trades", sprdId))
}

// Orders 撮合交易订单频道
func (w *PrivateClient) Orders(ctx context.Context, instType string, callback func(resp *WsResp[*Order])) error {
	if err := w.login(ctx); err != nil {
		return err
	}
	return Subscribe(&w.WsClient, ctx, makeArg("orders", instType), callback)
}

// UOrders 取消订阅撮合交易订单频道
func (w *PrivateClient) UOrders(instType string) error {
	return w.unsubscribe(makeArg("orders", instType))
}

// SpotOrders 撮合交易订单频道
func (w *PrivateClient) SpotOrders(ctx context.Context, callback func(resp *WsResp[*Order])) error {
	return Subscribe(&w.WsClient, ctx, makeArg("orders", "SPOT"), callback)
}

// USpotOrders 取消订阅撮合交易订单频道
func (w *PrivateClient) USpotOrders() error {
	return w.unsubscribe(makeArg("orders", "SPOT"))
}
