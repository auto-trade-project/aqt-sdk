package okx

import (
	"context"

	"github.com/kurosann/aqt-sdk/api/common"
)

type BusinessClient struct {
	common.WsClient
}

func (w *BusinessClient) MarkPriceCandlesticks(ctx context.Context, channel, instId string, callback func(resp *common.WsResp[*common.MarkPriceCandle])) error {
	if err := w.Login(ctx); err != nil {
		return err
	}
	return common.Subscribe(&w.WsClient, ctx, common.MakeArg("candle"+channel, instId), callback)
}
func (w *BusinessClient) UMarkPriceCandlesticks(channel, instId string) error {
	return w.Unsubscribe(common.MakeArg("mark-price-candle"+channel, instId))
}
func (w *BusinessClient) Candle(ctx context.Context, channel, instId string, callback func(resp *common.WsResp[*common.Candle])) error {
	return common.Subscribe(&w.WsClient, ctx, common.MakeArg("candle"+channel, instId), callback)
}
func (w *BusinessClient) UCandle(channel, instId string) error {
	return w.Unsubscribe(common.MakeArg("candle"+channel, instId))
}

type PublicClient struct {
	common.WsClient
}

func (w *PublicClient) MarkPrice(ctx context.Context, instId string, callback func(resp *common.WsResp[*common.MarkPrice])) error {
	return common.Subscribe(&w.WsClient, ctx, common.MakeArg("mark-price", instId), callback)
}
func (w *PublicClient) UMarkPrice(instId string) error {
	return w.Unsubscribe(common.MakeArg("mark-price", instId))
}

func (w *BusinessClient) OrderBook(ctx context.Context, channel, sprdId string, callback func(resp *common.WsResp[*common.OrderBook])) error {
	return common.Subscribe(&w.WsClient, ctx, common.MakeSprdArg(channel, sprdId), callback)
}
func (w *BusinessClient) UOrderBook(channel, instId string) error {
	return w.Unsubscribe(common.MakeSprdArg(channel, instId))
}

//-------------------------- 交易 --------------------------

type PrivateClient struct {
	common.WsClient
}

// Account 资金频道
func (w *PrivateClient) Account(ctx context.Context, callback func(resp *common.WsResp[*common.Balance])) error {
	if err := w.Login(ctx); err != nil {
		return err
	}
	return common.Subscribe(&w.WsClient, ctx, common.MakeArg("account", ""), callback)
}
func (w *PrivateClient) UAccount() error {
	return w.Unsubscribe(common.MakeArg("account", ""))
}

// Positions 持仓频道
func (w *PrivateClient) Positions(ctx context.Context, callback func(resp *common.WsResp[*common.Balances])) error {
	if err := w.Login(ctx); err != nil {
		return err
	}
	return common.Subscribe(&w.WsClient, ctx, common.MakeArg("positions", ""), callback)
}

// Trades 成交订单频道
func (w *BusinessClient) Trades(ctx context.Context, sprdId string, callback func(resp *common.WsResp[*common.Trades])) error {
	if err := w.Login(ctx); err != nil {
		return err
	}
	return common.Subscribe(&w.WsClient, ctx, common.MakeSprdArg("sprd-trades", sprdId), callback)
}

// UTrades 成交订单频道
func (w *BusinessClient) UTrades(sprdId string) error {
	return w.Unsubscribe(common.MakeSprdArg("sprd-trades", sprdId))
}

// Orders 撮合交易订单频道
func (w *PrivateClient) Orders(ctx context.Context, instType string, callback func(resp *common.WsResp[*common.Order])) error {
	if err := w.Login(ctx); err != nil {
		return err
	}
	return common.Subscribe(&w.WsClient, ctx, common.MakeArg("orders", instType), callback)
}

// UOrders 取消订阅撮合交易订单频道
func (w *PrivateClient) UOrders(instType string) error {
	return w.Unsubscribe(common.MakeArg("orders", instType))
}

// SpotOrders 撮合交易订单频道
func (w *PrivateClient) SpotOrders(ctx context.Context, callback func(resp *common.WsResp[*common.Order])) error {
	if err := w.Login(ctx); err != nil {
		return err
	}
	return common.Subscribe(&w.WsClient, ctx, &common.Arg{
		Channel:  "orders",
		InstType: "SPOT",
	}, callback)
}

// USpotOrders 取消订阅撮合交易订单频道
func (w *PrivateClient) USpotOrders() error {
	return w.Unsubscribe(common.MakeArg("orders", "SPOT"))
}
