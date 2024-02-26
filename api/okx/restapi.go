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
func (c RestClient) TakerVolume(ctx context.Context, req *TakerVolumeReq) (*Resp[TakerVolume], error) {
	return Get[TakerVolume](c, ctx, "/api/v5/rubik/stat/taker-volume", req)
}
func (c RestClient) Candles(ctx context.Context, req *CandlesticksReq) (*Resp[Candle], error) {
	return Get[Candle](c, ctx, "/api/v5/market/candles", req)
}
