package config

import "os"

type LivekitConfig struct {
	LiveKitHost string
	ApiKey      string
	ApiSecret   string
}

func LoadLivekitConfig() *LivekitConfig {
	return &LivekitConfig{
		LiveKitHost: os.Getenv("LIVEKIT_HOST"),
		ApiKey:      os.Getenv("LIVEKIT_API_KEY"),
		ApiSecret:   os.Getenv("LIVEKIT_API_SECRET"),
	}
}
