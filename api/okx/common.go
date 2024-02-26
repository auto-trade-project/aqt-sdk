package okx

import (
	"encoding/json"
	"errors"
	"fmt"
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
	Args []*Arg `json:"args"`
}
type WsResp struct {
	Event  string     `json:"event"`
	ConnId string     `json:"connId"`
	Code   string     `json:"code"`
	Msg    string     `json:"msg"`
	Arg    Arg        `json:"arg"`
	Data   RawMessage `json:"data"`
}
type PlaceOrderReq struct {
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
type Candles struct {
	InstId string `json:"instId"`
	Bar    string `json:"bar"`
	After  string `json:"after"`
	Before string `json:"before"`
	Limit  string `json:"limit"`
}

type Resp[T any] struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []T    `json:"data"`
}

func makeArg(channel, InstId string) *Arg {
	return &Arg{
		Channel: channel,
		InstId:  InstId,
	}
}
func makeOp(op string, args []*Arg) Op {
	return Op{
		Op:   op,
		Args: args,
	}
}

type PlaceOrder struct {
	ClOrdId string `json:"clOrdId"`
	OrdId   string `json:"ordId"`
	Tag     string `json:"tag"`
	SCode   string `json:"sCode"`
	SMsg    string `json:"sMsg"`
}

type Candle struct {
	Ts          string `json:"ts"`
	O           string `json:"o"`
	H           string `json:"h"`
	L           string `json:"l"`
	C           string `json:"c"`
	Vol         string `json:"vol"`
	VolCcy      string `json:"volCcy"`
	VolCcyQuote string `json:"volCcyQuote"`
	Confirm     string `json:"confirm"`
}

func (p *Candle) UnmarshalJSON(bytes []byte) (err error) {
	str, err := unmarshalSliceString(bytes)
	if err != nil {
		return err
	}
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			err = fmt.Errorf("price 协议格式不正确")
			return
		}
	}()
	p.Ts = str[0]
	p.O = str[1]
	p.H = str[2]
	p.L = str[3]
	p.C = str[4]
	p.Vol = str[5]
	p.VolCcy = str[6]
	p.VolCcyQuote = str[7]
	p.Confirm = str[8]
	return nil
}

type Instruments struct {
	Alias        string `json:"alias"`
	BaseCcy      string `json:"baseCcy"`
	Category     string `json:"category"`
	CtMult       string `json:"ctMult"`
	CtType       string `json:"ctType"`
	CtVal        string `json:"ctVal"`
	CtValCcy     string `json:"ctValCcy"`
	ExpTime      string `json:"expTime"`
	InstFamily   string `json:"instFamily"`
	InstId       string `json:"instId"`
	InstType     string `json:"instType"`
	Lever        string `json:"lever"`
	ListTime     string `json:"listTime"`
	LotSz        string `json:"lotSz"`
	MaxIcebergSz string `json:"maxIcebergSz"`
	MaxLmtAmt    string `json:"maxLmtAmt"`
	MaxLmtSz     string `json:"maxLmtSz"`
	MaxMktAmt    string `json:"maxMktAmt"`
	MaxMktSz     string `json:"maxMktSz"`
	MaxStopSz    string `json:"maxStopSz"`
	MaxTriggerSz string `json:"maxTriggerSz"`
	MaxTwapSz    string `json:"maxTwapSz"`
	MinSz        string `json:"minSz"`
	OptType      string `json:"optType"`
	QuoteCcy     string `json:"quoteCcy"`
	SettleCcy    string `json:"settleCcy"`
	State        string `json:"state"`
	Stk          string `json:"stk"`
	TickSz       string `json:"tickSz"`
	Uly          string `json:"uly"`
}

type TakerVolumeReq struct {
	Ccy      string `json:"ccy"`
	InstType string `json:"instType"`
	Begin    string `json:"begin"`
	End      string `json:"end"`
	Period   string `json:"period"`
}
type TakerVolume struct {
	Ts      string `json:"ts"`
	SellVol string `json:"sellVol"`
	BuyVol  string `json:"buyVol"`
}

type GetCandlesticks struct {
	InstID string `json:"instId"`
	After  int64  `json:"after,omitempty,string"`
	Before int64  `json:"before,omitempty,string"`
	Limit  int64  `json:"limit,omitempty,string"`
	Bar    string `json:"bar,omitempty"`
}

func (t *TakerVolume) UnmarshalJSON(bytes []byte) (err error) {
	str, err := unmarshalSliceString(bytes)
	if err != nil {
		return err
	}
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			err = fmt.Errorf("price 协议格式不正确")
			return
		}
	}()
	t.Ts = str[0]
	t.SellVol = str[1]
	t.BuyVol = str[2]
	return nil
}

type Price struct {
	Ts      string // 时间戳
	O       string // 开盘价格
	H       string // 最高价格
	L       string // 最低价格
	C       string // 收盘价格
	Confirm string // K线状态
}

func (p *Price) UnmarshalJSON(bytes []byte) (err error) {
	str, err := unmarshalSliceString(bytes)
	if err != nil {
		return err
	}
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			err = fmt.Errorf("price 协议格式不正确")
			return
		}
	}()
	p.Ts = str[0]
	p.O = str[1]
	p.H = str[2]
	p.L = str[3]
	p.C = str[4]
	p.Confirm = str[5]
	return nil
}

func unmarshalSliceString(data []byte) ([]string, error) {
	tmp, err := unmarshal[[]string](data)
	if err != nil {
		return nil, err
	}
	return *tmp, nil
}

func unmarshal[T any](data []byte) (*T, error) {
	t := new(T)
	if err := json.Unmarshal(data, t); err != nil {
		return nil, err
	}
	return t, nil
}

type MarkPrice struct {
	InstType string `json:"instType"`
	InstId   string `json:"instId"`
	MarkPx   string `json:"markPx"`
	Ts       string `json:"ts"`
}
type Spread struct {
	Price      string
	Count      string
	OrderCount string
}
type OrderBook struct {
	Asks []Spread `json:"asks"`
	Bids []Spread `json:"bids"`
	Ts   string   `json:"ts"`
}

func (o *OrderBook) UnmarshalJSON(bytes []byte) (err error) {
	var tmp = struct {
		Asks [][]string `json:"asks"`
		Bids [][]string `json:"bids"`
		Ts   string     `json:"ts"`
	}{}
	err = json.Unmarshal(bytes, &tmp)
	if err != nil {
		return err
	}
	defer func() {
		if recoverErr := recover(); recoverErr != nil {
			err = fmt.Errorf("spread 协议格式不正确")
			return
		}
	}()
	var asks []Spread
	for _, ask := range tmp.Asks {
		asks = append(asks, Spread{
			Price:      ask[0],
			Count:      ask[1],
			OrderCount: ask[2],
		})
	}
	var bids []Spread
	for _, bid := range tmp.Bids {
		bids = append(bids, Spread{
			Price:      bid[0],
			Count:      bid[1],
			OrderCount: bid[2],
		})
	}
	o.Asks = asks
	o.Bids = bids
	o.Ts = tmp.Ts
	return nil
}
