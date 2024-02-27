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
			"e1fd21f8-f934-4cf7-9921-b747f77fe526",
			"607295DE8672908D2D6A46F2774550FD",
			"14383421@Qq",
		},
		TestServer,
	)
	count := 2
	cond := sync.NewCond(&sync.RWMutex{})
	go func() {
		ch, err := client.Account()
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
		<-time.After(time.Second * 20)

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
