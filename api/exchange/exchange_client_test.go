package exchange

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kurosann/aqt-sdk/api"
	"github.com/kurosann/aqt-sdk/api/okx"
)

var config = okx.OkxKeyConfig{
	"",
	"",
	"",
}

func newClient() api.IMarketClient {
	client, _ := NewMarketApi(context.Background(),
		okx.UseOkxExchange(
			okx.WithConfig(config),
			okx.WithEnv(okx.NormalServer),
		),
	)
	return client
}
func TestNewWsClient(t *testing.T) {
	client := newClient()
	count := 200
	cond := sync.NewCond(&sync.RWMutex{})
	go func() {
		if err := client.CandleListen(context.Background(), time.Minute*15, "JUP-USDT", func(resp *api.Candle) {
			t.Log(resp)
			cond.L.Lock()
			count--
			cond.Broadcast()
			cond.L.Unlock()
		}); err != nil {
			assert.Fail(t, err.Error())
			return
		}
		//if err := client.MarkPriceListen("SOL-USDT", func(resp *WsResp[*MarkPriceListen]) {
		//	t.Log(*resp)
		//	cond.L.Lock()
		//	count--
		//	cond.Broadcast()
		//	cond.L.Unlock()
		//}); err != nil {
		//	assert.Fail(t, err.Error())
		//	return
		//}
	}()
	//go func() {
	//	ch, err := client.MarkPriceListen("BTC-USDT")
	//	if err != nil {
	//		assert.Fail(t, err.Error())
	//		return
	//	}
	//	for {
	//		resp, ok := <-ch
	//		if !ok {
	//			return
	//		}
	//		fmt.Println(*resp)
	//		cond.L.Lock()
	//		count--
	//		cond.Broadcast()
	//		cond.L.Unlock()
	//	}
	//}()
	isFailed := false
	go func() {
		<-time.After(time.Second * 200000)

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
