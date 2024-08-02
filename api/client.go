package api

import (
	"context"
	"fmt"

	"github.com/kurosann/aqt-sdk/api/okx"
)

func NewMarketApi(ctx context.Context, opt OptInfo) (exchange IMarketApi, err error) {
	switch opt.Exchange {
	case OkxExchange:
		exchange, err = okx.NewExchangeClient(ctx, opt.Opts...)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("no support %s", opt.Exchange)
	}
	return exchange, nil
}
