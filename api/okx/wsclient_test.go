package okx

import (
	"context"
	"fmt"
	"testing"
)

func TestNewWsClient(t *testing.T) {
	client := NewWsClient(
		context.Background(),
		"bf8723fa-3070-46b8-b964-2f68c0a9320c",
		"7EA20DF3B56F713E00E43EF84E910CD3",
		"",
		TestServer,
	)
	go func() {
		ch, err := client.MarkPriceCandlesticks("mark-price-candle1M", "BTC-USDT")
		if err != nil {
			t.Log(err)
			return
		}
		for {
			resp, ok := <-ch
			if !ok {
				return
			}
			fmt.Println(*resp)
		}
	}()
	go func() {
		ch, err := client.MarkPrice("BTC-USDT")
		if err != nil {
			t.Log(err)
			return
		}
		for {
			resp, ok := <-ch
			if !ok {
				return
			}
			fmt.Println(*resp)
		}
	}()
	select {}
}
