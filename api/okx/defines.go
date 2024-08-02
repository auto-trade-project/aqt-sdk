package okx

type NormType string

var (
	M15 = NormType("15m")
	H1  = NormType("1H")
	D1  = NormType("1D")
)

type EnvType string

var (
	OkxType     = EnvType("Okx")
	BinanceType = EnvType("Binance")
)

var channelOkxMap = map[NormType]string{
	M15: "mark-price-candle15m",
	H1:  "mark-price-candle1H",
	D1:  "mark-price-candle1D",
}
