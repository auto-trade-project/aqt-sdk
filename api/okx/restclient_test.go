package okx

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRestClient(t *testing.T) {
	client := NewRestClient(context.Background(), config, TestServer)
	rp, err := client.Instruments(context.Background(), InstrumentsReq{
		InstType: "SPOT",
	})
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}
	fmt.Println(rp)
}

func TestHistoryMarkPriceCandles(t *testing.T) {
	client := NewRestClient(context.Background(), config, TestServer)
	today := time.Now()
	todayZero := time.Date(today.Year(), today.Month(), today.Day(), today.Hour(), 0, 0, 0, today.Location())
	var startTime = todayZero.Add(time.Duration(-3*24) * time.Hour).UnixMilli()

	rp, err := client.HistoryMarkPriceCandles(context.Background(), MarkPriceCandlesReq{
		InstID: "BTC-USDT",
		After:  time.Now().UnixMilli(),
		Before: startTime,
		Bar:    "15m",
		Limit:  100,
	})
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}
	fmt.Println(rp)
}

func TestGetCandlesticks(t *testing.T) {
	client := NewRestClient(context.Background(), config, TestServer)
	today := time.Now()
	todayZero := time.Date(today.Year(), today.Month(), today.Day(), today.Hour(), 0, 0, 0, today.Location())
	var startTime = todayZero.Add(time.Duration(-3*24) * time.Hour).UnixMilli()
	rp, err := client.Candles(
		context.Background(),
		CandlesticksReq{
			InstID: "BTC-USDT",
			After:  time.Now().UnixMilli(),
			Before: startTime,
			Bar:    "15m",
			Limit:  100,
		})
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}
	fmt.Println(rp)
}
