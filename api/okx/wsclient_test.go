package okx

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

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
	count := 10
	cond := sync.NewCond(&sync.RWMutex{})
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
			cond.L.Lock()
			count--
			cond.Broadcast()
			cond.L.Unlock()
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
			cond.L.Lock()
			count--
			cond.Broadcast()
			cond.L.Unlock()
		}
	}()
	isFailed := false
	go func() {
		<-time.After(time.Second * 10)

		cond.L.Lock()
		isFailed = true
		cond.Broadcast()
		cond.L.Unlock()
	}()
	cond.L.Lock()
	for count != 0 && !isFailed {
		cond.Wait()
	}
	cond.L.Unlock()
	assert.Equal(t, count, 0)
}
