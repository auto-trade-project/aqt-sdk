package okx

import (
	"context"
)

func (c RestClient) PlaceOrder(ctx context.Context, req PlaceOrderReq) (*Resp[PlaceOrder], error) {
	data, err := c.Get(ctx, "/api/v5/trade/order", req)
	if err != nil {
		return nil, err
	}
	return unmarshal[Resp[PlaceOrder]](data)
}
func (c RestClient) Instruments(ctx context.Context, instType string) (*Resp[Instruments], error) {
	data, err := c.Post(ctx, "/api/v5/public/instruments", map[string]interface{}{
		"instType": instType,
	})
	if err != nil {
		return nil, err
	}
	return unmarshal[Resp[Instruments]](data)
}
func (c RestClient) HistoryMarkPriceCandles(ctx context.Context, req *Candles) (*Resp[Candle], error) {
	data, err := c.Get(ctx, "/api/v5/market/history-mark-price-candles", req)
	if err != nil {
		return nil, err
	}
	return unmarshal[Resp[Candle]](data)
}
func (c RestClient) TakerVolume(ctx context.Context, req *TakerVolumeReq) (*Resp[TakerVolume], error) {
	data, err := c.Get(ctx, "/api/v5/rubik/stat/taker-volume", req)
	if err != nil {
		return nil, err
	}
	return unmarshal[Resp[TakerVolume]](data)
}
