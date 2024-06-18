package okx

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kurosann/aqt-sdk/api/common"
)

func TestNewRestClient(t *testing.T) {
	client := NewRestClient(context.Background(), config, common.TestServer)
	rp, err := client.Instruments(context.Background(), common.InstrumentsReq{
		InstType: "SPOT",
	})
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}
	fmt.Println(rp)
}

func TestHistoryMarkPriceCandles(t *testing.T) {
	client := NewRestClient(context.Background(), config, common.TestServer)
	today := time.Now()
	todayZero := time.Date(today.Year(), today.Month(), today.Day(), today.Hour(), 0, 0, 0, today.Location())
	var startTime = todayZero.Add(time.Duration(-3*24) * time.Hour).UnixMilli()

	rp, err := client.HistoryMarkPriceCandles(context.Background(), common.MarkPriceCandlesReq{
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
	client := NewRestClient(context.Background(), config, common.TestServer)
	today := time.Now()
	todayZero := time.Date(today.Year(), today.Month(), today.Day(), today.Hour(), 0, 0, 0, today.Location())
	var startTime = todayZero.Add(time.Duration(-3*24) * time.Hour).UnixMilli()
	rp, err := client.Candles(
		context.Background(),
		common.CandlesticksReq{
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
