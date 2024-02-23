package okx

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRestClient(t *testing.T) {
	client := NewRestClient(context.Background(), KeyConfig{
		"bf8723fa-3070-46b8-b964-2f68c0a9320c",
		"7EA20DF3B56F713E00E43EF84E910CD3",
		"",
	}, TestServer)
	rp, err := client.Instruments(context.Background(), "SPOT")
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}
	bs, err := io.ReadAll(rp.Body)
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}
	assert.Equal(t, rp.StatusCode, 200, string(bs))
	fmt.Println(string(bs))
}
