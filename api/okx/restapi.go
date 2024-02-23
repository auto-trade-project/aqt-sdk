package okx

import (
	"context"
	"net/http"
)

func (c RestClient) PlaceOrder(ctx context.Context, req PlaceOrder) (*http.Response, error) {
	return c.Get(ctx, "/api/v5/trade/order", req)
}
func (c RestClient) Instruments(ctx context.Context, instType string) (*http.Response, error) {
	return c.Post(ctx, "/api/v5/public/instruments", map[string]interface{}{
		"instType": instType,
	})
}
