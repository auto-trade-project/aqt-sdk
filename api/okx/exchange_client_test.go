package okx

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kurosann/aqt-sdk/api"
)

var config = OkxKeyConfig{
	"",
	"",
	"",
}

func TestNewWsClient(t *testing.T) {
	client, _ := NewExchangeClient(
		context.Background(),
		WithConfig(config),
		WithEnv(TestServer),
	)
	count := 200
	cond := sync.NewCond(&sync.RWMutex{})
	go func() {
		if err := client.Account(context.Background(), func(resp api.Balance) {
			t.Log(resp)
			cond.L.Lock()
			count--
			cond.Broadcast()
			cond.L.Unlock()
		}); err != nil {
			assert.Fail(t, err.Error())
			return
		}
		//if err := client.MarkPrice("SOL-USDT", func(resp *WsResp[*MarkPrice]) {
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
	//	ch, err := client.MarkPrice("BTC-USDT")
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
