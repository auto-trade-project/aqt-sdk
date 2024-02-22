package okx

import (
	"context"
	"reflect"
	"testing"
)

func TestNewRestClient(t *testing.T) {
	type args struct {
		ctx       context.Context
		keyConfig KeyConfig
		env       Destination
	}
	tests := []struct {
		name string
		args args
		want *RestClient
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewRestClient(tt.args.ctx, tt.args.keyConfig, tt.args.env); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRestClient() = %v, want %v", got, tt.want)
			}
		})
	}
}
