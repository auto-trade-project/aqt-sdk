package okx

func (w *WsClient) MarkPriceCandlesticks(channel, instId string) (<-chan *WsResp, error) {
	return w.Subscribe(makeArg(channel, instId), Business)
}

func (w *WsClient) UMarkPriceCandlesticks(channel, instId string) error {
	return w.UnSubscribe(makeArg(channel, instId), Business)
}
func (w *WsClient) MarkPrice(instId string) (<-chan *WsResp, error) {
	return w.Subscribe(makeArg("mark-price", instId), Public)
}
func (w *WsClient) UMarkPrice(instId string) error {
	return w.UnSubscribe(makeArg("mark-price", instId), Public)
}
func (w *WsClient) OrderBook(channel, instId string) (<-chan *WsResp, error) {
	return w.Subscribe(makeArg(channel, instId), Business)
}
func (w *WsClient) UOrderBook(channel, instId string) error {
	return w.UnSubscribe(makeArg(channel, instId), Business)
}
