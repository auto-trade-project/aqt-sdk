package okx

import "fmt"

func MatchChannel(env EnvType, norm NormType) string {
	channels := matchByEnv(env)
	return channels[norm]
}

func matchByEnv(env EnvType) map[NormType]string {
	switch env {
	case OkxType:
		return channelOkxMap
	default:
		return channelOkxMap
	}
}

func MatchNormType(env EnvType, channelName string) (NormType, error) {
	channels := matchByEnv(env)
	for n, v := range channels {
		if channelName == v {
			return n, nil
		}
	}
	return GetNormType(env, channelName)
}

func GetNormType(env EnvType, name string) (NormType, error) {
	channels := matchByEnv(env)
	for n := range channels {
		if string(n) == name {
			return n, nil
		}
	}
	return NormType(""), fmt.Errorf("not found norm type by name %s", name)
}
