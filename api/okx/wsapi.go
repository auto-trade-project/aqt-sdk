package okx

func (w *WsClient) MarkPriceCandlesticks(channel, instId string) (<-chan *WsResp, error) {
	return w.Subscribe(makeArg(channel, instId, nil), Business)
}

func (w *WsClient) UMarkPriceCandlesticks(channel, instId string) error {
	return w.UnSubscribe(makeArg(channel, instId, nil), Business)
}
func (w *WsClient) MarkPrice(instId string) (<-chan *WsResp, error) {
	return w.Subscribe(makeArg("mark-price", instId, nil), Public)
}
func (w *WsClient) UMarkPrice(instId string) error {
	return w.UnSubscribe(makeArg("mark-price", instId, nil), Public)
}
func (w *WsClient) OrderBook(channel, instId string) (<-chan *WsResp, error) {
	return w.Subscribe(makeArg(channel, instId, nil), Business)
}
func (w *WsClient) UOrderBook(channel, instId string) error {
	return w.UnSubscribe(makeArg(channel, instId, nil), Business)
}

func (w *WsClient) Login() (chan *WsResp, error) {
	ch := make(chan *WsResp)
	if respCh, isExist := w.RegCh("login", ch); isExist {
		ch = respCh
	}
	return ch, w.Send(Private, Op{
		Op:   "login",
		Args: []map[string]string{w.keyConfig.makeWsSign()},
	})
}

//-------------------------- 交易 --------------------------

// Account 资金频道
func (w *WsClient) Account() (<-chan *WsResp, error) {
	return w.Subscribe(makeArg("account", "", nil), Private)
}
func (w *WsClient) UAccount() error {
	return w.UnSubscribe(makeArg("account", "", nil), Private)
}

// Positions 持仓频道
func (w *WsClient) Positions() (<-chan *WsResp, error) {
	return w.Subscribe(makeArg("positions", "", nil), Private)
}
