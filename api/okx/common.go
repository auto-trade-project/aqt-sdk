package okx

import (
	"encoding/json"
	"errors"
)

type (
	Destination int
	BaseURL     string
)

const (
	NormalServer Destination = iota
	AwsServer
	TestServer

	RestURL       = BaseURL("https://www.okx.com")
	PublicWsURL   = BaseURL("wss://ws.okx.com:8443/ws/v5/public")
	PrivateWsURL  = BaseURL("wss://ws.okx.com:8443/ws/v5/private")
	BusinessWsURL = BaseURL("wss://ws.okx.com:8443/ws/v5/business?brokerId=9999")

	AwsRestURL       = BaseURL("https://aws.okx.com")
	AwsPublicWsURL   = BaseURL("wss://wsaws.okx.com:8443/ws/v5/public")
	AwsPrivateWsURL  = BaseURL("wss://wsaws.okx.com:8443/ws/v5/private")
	AwsBusinessWsURL = BaseURL("wss://wsaws.okx.com:8443/ws/v5/business?brokerId=9999")

	TestRestURL       = BaseURL("https://www.okx.com")
	TestPublicWsURL   = BaseURL("wss://wspap.okx.com:8443/ws/v5/public?brokerId=9999")
	TestPrivateWsURL  = BaseURL("wss://wspap.okx.com:8443/ws/v5/private?brokerId=9999")
	TestBusinessWsURL = BaseURL("wss://wspap.okx.com:8443/ws/v5/business?brokerId=9999")
)

var (
	DefaultWsUrls = map[Destination]map[SvcType]BaseURL{
		NormalServer: {
			Public:   PublicWsURL,
			Private:  PrivateWsURL,
			Business: BusinessWsURL,
		},
		AwsServer: {
			Public:   AwsPublicWsURL,
			Private:  AwsPrivateWsURL,
			Business: AwsBusinessWsURL,
		},
		TestServer: {
			Public:   TestPublicWsURL,
			Private:  TestPrivateWsURL,
			Business: TestBusinessWsURL,
		},
	}
	DefaultRestUrl = map[Destination]BaseURL{
		NormalServer: RestURL,
		AwsServer:    AwsRestURL,
		TestServer:   TestRestURL,
	}
)

type RawMessage []byte

func (r RawMessage) String() string {
	return string(r)
}
func (r RawMessage) Unmarshal(any any) error {
	return json.Unmarshal(r, any)
}

// MarshalJSON returns m as the JSON encoding of m.
func (m RawMessage) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return m, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (m *RawMessage) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}

type KeyConfig struct {
	Apikey     string
	Secretkey  string
	Passphrase string
}
type SubChannel struct {
	Channel string `json:"channel"`
	InstId  string `json:"instId"`
}
type UnSubChannel struct {
	Channel string `json:"channel"`
	InstId  string `json:"instId"`
}
type Arg struct {
	Channel string `json:"channel"`
	InstId  string `json:"instId"`
}
type Op struct {
	Op   string `json:"op"`
	Args []Arg  `json:"args"`
}
type WsResp struct {
	Event  string     `json:"event"`
	ConnId string     `json:"connId"`
	Code   string     `json:"code"`
	Msg    string     `json:"msg"`
	Arg    Arg        `json:"arg"`
	Data   RawMessage `json:"data"`
}
type PlaceOrder struct {
	ID         string  `json:"-"`
	InstID     string  `json:"instId"`
	Ccy        string  `json:"ccy,omitempty"`
	ClOrdID    string  `json:"clOrdId,omitempty"`
	Tag        string  `json:"tag,omitempty"`
	ReduceOnly bool    `json:"reduceOnly,omitempty"`
	Sz         float64 `json:"sz,string"`
	Px         float64 `json:"px,omitempty,string"`
	TdMode     string  `json:"tdMode"`
	Side       string  `json:"side"`
	PosSide    string  `json:"posSide,omitempty"`
	OrdType    string  `json:"ordType"`
	TgtCcy     string  `json:"tgtCcy,omitempty"`
}

func makeArg(channel, InstId string) Arg {
	return Arg{
		Channel: channel,
		InstId:  InstId,
	}
}
func makeOp(op string, args []Arg) Op {
	return Op{
		Op:   op,
		Args: args,
	}
}
