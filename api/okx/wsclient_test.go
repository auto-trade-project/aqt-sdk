package okx

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWsClient(t *testing.T) {
	client := NewWsClient(
		context.Background(),
		KeyConfig{
			"",
			"",
			"",
		},
		TestServer,
	)
	group := sync.WaitGroup{}
	group.Add(10)
	go func() {
		ch, err := client.MarkPriceCandlesticks("mark-price-candle1M", "BTC-USDT")
		if err != nil {
			assert.Fail(t, err.Error())
			return
		}
		for {
			resp, ok := <-ch
			if !ok {
				return
			}
			fmt.Println(*resp)
			group.Done()
		}
	}()
	go func() {
		ch, err := client.MarkPrice("BTC-USDT")
		if err != nil {
			assert.Fail(t, err.Error())
			return
		}
		for {
			resp, ok := <-ch
			if !ok {
				return
			}
			fmt.Println(*resp)
			group.Done()
		}
	}()
	group.Wait()
}
