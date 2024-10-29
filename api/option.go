package api

func WithLogger(log ILogger) Opt {
	return func(a IMarketClient) {
		if client, ok := a.(ISetLogger); ok {
			client.SetLog(log)
		}
	}
}
func WithReadMonitor(readMonitor func(arg string)) Opt {
	return func(api IMarketClient) {
		if client, ok := api.(IReadMonitor); ok {
			client.ReadMonitor(readMonitor)
		}
	}
}
