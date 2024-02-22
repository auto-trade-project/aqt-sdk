package okx

func (w *WsClient) MarkPriceCandlesticks(channel, InstId string) (<-chan *WsResp, error) {
	return w.Subscribe(channel, InstId, Business)
}

func (w *WsClient) UMarkPriceCandlesticks(channel, InstId string) (<-chan *WsResp, error) {
	return w.UnSubscribe(channel, InstId, Business)
}
func (w *WsClient) MarkPrice(InstId string) (<-chan *WsResp, error) {
	return w.Subscribe("mark-price", InstId, Public)
}

func (w *WsClient) UMarkPrice(InstId string) (<-chan *WsResp, error) {
	return w.UnSubscribe("mark-price", InstId, Public)
}
func (w *WsClient) OrderBook(channel, InstId string) (<-chan *WsResp, error) {
	return w.Subscribe(channel, InstId, Public)
}
func (w *WsClient) UOrderBook(channel, InstId string) (<-chan *WsResp, error) {
	return w.UnSubscribe(channel, InstId, Public)
}
