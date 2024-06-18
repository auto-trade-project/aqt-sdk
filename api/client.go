package api

import (
	"context"
	"fmt"

	"github.com/kurosann/aqt-sdk/api/common"
	"github.com/kurosann/aqt-sdk/api/okx"
)

type ExClient interface {
	common.ClientBase
	Account(ctx context.Context, callback func(resp *common.WsResp[*common.Balance])) error
	UAccount() error
	Candle(ctx context.Context, channel, instId string, callback func(resp *common.WsResp[*common.Candle])) error
	UCandle(channel, instId string) error
	MarkPrice(ctx context.Context, instId string, callback func(resp *common.WsResp[*common.MarkPrice])) error
	UMarkPrice(instId string) error
	SpotOrders(ctx context.Context, callback func(resp *common.WsResp[*common.Order])) error
	USpotOrders() error
}

var (
	clientMap = map[string]func(ctx context.Context, keyConfig common.IKeyConfig, env common.Destination, proxy ...string) ExClient{
		"okx": func(ctx context.Context, keyConfig common.IKeyConfig, env common.Destination, proxy ...string) ExClient {
			return okx.NewWsClient(ctx, keyConfig, env, proxy...)
		},
	}
)

// NewClient no concurrent safe
func NewClient(ctx context.Context, plm string, keyConfig common.IKeyConfig, env common.Destination, proxy ...string) (ExClient, error) {
	f, ok := clientMap[plm]
	if !ok {
		return nil, fmt.Errorf("no support %s", plm)
	}
	client := f(ctx, keyConfig, env, proxy...)
	return client, nil
}
