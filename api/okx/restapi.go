package okx

import (
	"context"

	"github.com/kurosann/aqt-sdk/api/common"
)

// PlaceOrder 下单
func (c *RestClient) PlaceOrder(ctx context.Context, req common.PlaceOrderReq) (*common.Resp[common.PlaceOrder], error) {
	return Post[common.PlaceOrder](c, ctx, "/api/v5/trade/order", req)
}

// CancelOrder 取消挂单信息
func (c *RestClient) CancelOrder(ctx context.Context, instId, clOrdId string) (*common.Resp[common.PlaceOrder], error) {
	return Get[common.PlaceOrder](c, ctx, "/api/v5/trade/cancel-order", map[string]string{
		"clOrdId": clOrdId,
		"instId":  instId,
	})
}

// GetOrder 获取订单信息
func (c *RestClient) GetOrder(ctx context.Context, req common.PlaceOrderReq) (*common.Resp[common.Order], error) {
	return Get[common.Order](c, ctx, "/api/v5/trade/order", req)
}

// Instruments 获取产品信息
func (c *RestClient) Instruments(ctx context.Context, req common.InstrumentsReq) (*common.Resp[common.Instruments], error) {
	return Get[common.Instruments](c, ctx, "/api/v5/public/instruments", req)
}

// MarkPriceCandles 获取当前k线标价
func (c *RestClient) MarkPriceCandles(ctx context.Context, req common.MarkPriceCandlesReq) (*common.Resp[common.MarkPriceCandle], error) {
	return Get[common.MarkPriceCandle](c, ctx, "/api/v5/market/mark-price-candles", req)
}

// HistoryMarkPriceCandles 获取历史k线标价
func (c *RestClient) HistoryMarkPriceCandles(ctx context.Context, req common.MarkPriceCandlesReq) (*common.Resp[common.MarkPriceCandle], error) {
	return Get[common.MarkPriceCandle](c, ctx, "/api/v5/market/history-mark-price-candles", req)
}

// TakerVolume 主动买卖交易量
func (c *RestClient) TakerVolume(ctx context.Context, req common.TakerVolumeReq) (*common.Resp[common.TakerVolume], error) {
	return Get[common.TakerVolume](c, ctx, "/api/v5/rubik/stat/taker-volume", req)
}

// Candles 获取产品k线标价
func (c *RestClient) Candles(ctx context.Context, req common.CandlesticksReq) (*common.Resp[common.Candle], error) {
	return Get[common.Candle](c, ctx, "/api/v5/market/candles", req)
}

// Balances 资产账户余额
func (c *RestClient) Balances(ctx context.Context, ccy string) (*common.Resp[common.Balances], error) {
	return Get[common.Balances](c, ctx, "/api/v5/asset/balances", map[string]string{
		"ccy": ccy,
	})
}

// LoanRatio 获取多空比
func (c *RestClient) LoanRatio(ctx context.Context, ccy, begin, end, period string) (*common.Resp[[]string], error) {
	return Get[[]string](c, ctx, "/api/v5/rubik/stat/margin/loan-ratio", map[string]string{
		"ccy":    ccy,
		"begin":  begin,
		"end":    end,
		"period": period,
	})
}

// LongShortAccountRatio 获取合约多空持仓人数比
func (c *RestClient) LongShortAccountRatio(ctx context.Context, ccy, begin, end, period string) (*common.Resp[[]string], error) {
	return Get[[]string](c, ctx, "/api/v5/rubik/stat/contracts/long-short-account-ratio", map[string]string{
		"ccy":    ccy,
		"begin":  begin,
		"end":    end,
		"period": period,
	})
}

//-------------------------- 交易 --------------------------

// Balance 交易账户余额
func (c *RestClient) Balance(ctx context.Context, ccy string) (*common.Resp[common.Balance], error) {
	return Get[common.Balance](c, ctx, "/api/v5/account/balance", map[string]string{
		"ccy": ccy,
	})
}

// Positions 账户持仓信息
func (c *RestClient) Positions(ctx context.Context, req common.PositionReq) (*common.Resp[common.Balances], error) {
	return Get[common.Balances](c, ctx, "/api/v5/account/positions", req)
}
