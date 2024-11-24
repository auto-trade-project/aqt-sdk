package exchange

import (
	"context"
	"fmt"

	"github.com/kurosann/aqt-sdk/api"
	"github.com/kurosann/aqt-sdk/api/okx"
)

func NewMarketApi(ctx context.Context, opt api.OptInfo) (exchange api.IMarketClient, err error) {
	switch opt.Exchange {
	case api.OkxExchange:
		exchange, err = okx.NewExchangeClient(ctx, opt.Opts...)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("no support %s", opt.Exchange)
	}
	return exchange, nil
}
