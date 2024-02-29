package okx

func (w *WsClient) MarkPriceCandlesticks(channel, instId string) (<-chan *WsResp, error) {
	return w.Subscribe(makeArg("mark-price-candle"+channel, instId), Business, false)
}
func (w *WsClient) UMarkPriceCandlesticks(channel, instId string) error {
	return w.UnSubscribe(makeArg("mark-price-candle"+channel, instId), Business)
}
func (w *WsClient) Candle(channel, instId string) (<-chan *WsResp, error) {
	return w.Subscribe(makeArg("candle"+channel, instId), Business, false)
}
func (w *WsClient) UCandle(channel, instId string) error {
	return w.UnSubscribe(makeArg("candle"+channel, instId), Business)
}
func (w *WsClient) MarkPrice(instId string) (<-chan *WsResp, error) {
	return w.Subscribe(makeArg("mark-price", instId), Public, false)
}
func (w *WsClient) UMarkPrice(instId string) error {
	return w.UnSubscribe(makeArg("mark-price", instId), Public)
}

func (w *WsClient) OrderBook(channel, sprdId string) (<-chan *WsResp, error) {
	return w.Subscribe(makeSprdArg(channel, sprdId), Business, false)
}
func (w *WsClient) UOrderBook(channel, instId string) error {
	return w.UnSubscribe(makeSprdArg(channel, instId), Business)
}

func (w *WsClient) Login(typ SvcType) (chan *WsResp, error) {
	ch := make(chan *WsResp)
	if respCh, isExist := w.RegCh("login", ch); isExist {
		ch = respCh
	}
	return ch, w.Send(typ, Op{
		Op:   "login",
		Args: []map[string]string{w.keyConfig.makeWsSign()},
	})
}

//-------------------------- 交易 --------------------------

// Account 资金频道
func (w *WsClient) Account() (<-chan *WsResp, error) {
	return w.Subscribe(makeArg("account", ""), Private, true)
}
func (w *WsClient) UAccount() error {
	return w.UnSubscribe(makeArg("account", ""), Private)
}

// Positions 持仓频道
func (w *WsClient) Positions() (<-chan *WsResp, error) {
	return w.Subscribe(makeArg("positions", ""), Private, true)
}

// Trades 成交订单频道
func (w *WsClient) Trades(sprdId string) (<-chan *WsResp, error) {
	return w.Subscribe(makeSprdArg("sprd-trades", sprdId), Business, true)
}
