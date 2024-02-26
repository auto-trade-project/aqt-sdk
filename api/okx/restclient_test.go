package okx

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRestClient(t *testing.T) {
	client := NewRestClient(context.Background(), KeyConfig{
		"",
		"",
		"",
	}, TestServer)
	rp, err := client.Instruments(context.Background(), InstrumentsReq{
		InstType: "SPOT",
	})
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}
	fmt.Println(rp)
}
