package okx

func (w *WsClient) MarkPriceCandlesticks(channel, instId string, callback func(resp *WsResp[*MarkPriceCandle])) error {
	return Subscribe(w, makeArg("mark-price-candle"+channel, instId), Business, callback, false)
}
func (w *WsClient) UMarkPriceCandlesticks(channel, instId string) error {
	return w.UnSubscribe(makeArg("mark-price-candle"+channel, instId), Business)
}
func (w *WsClient) Candle(channel, instId string, callback func(resp *WsResp[Candle])) error {
	return Subscribe(w, makeArg("candle"+channel, instId), Business, callback, false)
}
func (w *WsClient) UCandle(channel, instId string) error {
	return w.UnSubscribe(makeArg("candle"+channel, instId), Business)
}
func (w *WsClient) MarkPrice(instId string, callback func(resp *WsResp[*MarkPrice])) error {
	return Subscribe(w, makeArg("mark-price", instId), Public, callback, false)
}
func (w *WsClient) UMarkPrice(instId string) error {
	return w.UnSubscribe(makeArg("mark-price", instId), Public)
}

func (w *WsClient) OrderBook(channel, sprdId string, callback func(resp *WsResp[OrderBook])) error {
	return Subscribe(w, makeSprdArg(channel, sprdId), Business, callback, false)
}
func (w *WsClient) UOrderBook(channel, instId string) error {
	return w.UnSubscribe(makeSprdArg(channel, instId), Business)
}

func (w *WsClient) Login(typ SvcType, callback func(resp *WsOriginResp)) error {
	w.conns[typ].RegCallback(Arg{
		Channel: "login",
	}.Key(), callback)
	return w.Send(typ, Op{
		Op:   "login",
		Args: []map[string]string{w.keyConfig.makeWsSign()},
	})
}

//-------------------------- 交易 --------------------------

// Account 资金频道
func (w *WsClient) Account(callback func(resp *WsResp[Balance])) error {
	return Subscribe(w, makeArg("account", ""), Private, callback, true)
}
func (w *WsClient) UAccount() error {
	return w.UnSubscribe(makeArg("account", ""), Private)
}

// Positions 持仓频道
func (w *WsClient) Positions(callback func(resp *WsResp[Balances])) error {
	return Subscribe(w, makeArg("positions", ""), Private, callback, true)
}

// Trades 成交订单频道
func (w *WsClient) Trades(sprdId string, callback func(resp *WsResp[Trades])) error {
	return Subscribe(w, makeSprdArg("sprd-trades", sprdId), Business, callback, true)
}

// UTrades 成交订单频道
func (w *WsClient) UTrades(sprdId string) error {
	return w.UnSubscribe(makeSprdArg("sprd-trades", sprdId), Business)
}

// Orders 撮合交易订单频道
func (w *WsClient) Orders(instType string, callback func(resp *WsResp[Order])) error {
	return Subscribe(w, &Arg{
		Channel:  "orders",
		InstType: instType,
	}, Private, callback, true)
}

// UOrders 取消订阅撮合交易订单频道
func (w *WsClient) UOrders(instType string) error {
	return w.UnSubscribe(&Arg{
		Channel:  "orders",
		InstType: instType,
	}, Private)
}

// SpotOrders 撮合交易订单频道
func (w *WsClient) SpotOrders(callback func(resp *WsResp[Order])) error {
	return Subscribe(w, &Arg{
		Channel:  "orders",
		InstType: "SPOT",
	}, Private, callback, true)
}

// USpotOrders 取消订阅撮合交易订单频道
func (w *WsClient) USpotOrders() error {
	return w.UnSubscribe(&Arg{
		Channel:  "orders",
		InstType: "SPOT",
	}, Private)
}
