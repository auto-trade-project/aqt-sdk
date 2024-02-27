package okx

import (
	"context"
)

func (c RestClient) PlaceOrder(ctx context.Context, req PlaceOrderReq) (*Resp[PlaceOrder], error) {
	return Get[PlaceOrder](c, ctx, "/api/v5/trade/order", req)
}
func (c RestClient) Instruments(ctx context.Context, req InstrumentsReq) (*Resp[Instruments], error) {
	return Get[Instruments](c, ctx, "/api/v5/public/instruments", req)
}
func (c RestClient) HistoryMarkPriceCandles(ctx context.Context, req HistoryMarkPriceCandlesReq) (*Resp[Price], error) {
	return Get[Price](c, ctx, "/api/v5/market/history-mark-price-candles", req)
}
func (c RestClient) TakerVolume(ctx context.Context, req TakerVolumeReq) (*Resp[TakerVolume], error) {
	return Get[TakerVolume](c, ctx, "/api/v5/rubik/stat/taker-volume", req)
}
func (c RestClient) Candles(ctx context.Context, req CandlesticksReq) (*Resp[Candle], error) {
	return Get[Candle](c, ctx, "/api/v5/market/candles", req)
}

// Balances 资产账户余额
func (c RestClient) Balances(ctx context.Context, ccy string) (*Resp[Balances], error) {
	return Get[Balances](c, ctx, "/api/v5/asset/balances", map[string]string{
		"ccy": ccy,
	})
}

//-------------------------- 交易 --------------------------

// Balance 交易账户余额
func (c RestClient) Balance(ctx context.Context, ccy string) (*Resp[Balance], error) {
	return Get[Balance](c, ctx, "/api/v5/account/balance", map[string]string{
		"ccy": ccy,
	})
}

// Positions 账户持仓信息
func (c RestClient) Positions(ctx context.Context, req PositionReq) (*Resp[Balances], error) {
	return Get[Balances](c, ctx, "/api/v5/account/positions", req)
}
