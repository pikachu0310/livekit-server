package config

import (
	traqwsbot "github.com/traPtitech/traq-ws-bot"
)

func NewTraQBot() *traqwsbot.Bot {
	newBot, err := traqwsbot.NewBot(&traqwsbot.Options{
		AccessToken: getEnv("TRAQ_ACCESS_TOKEN", ""),
	})
	if err != nil {
		panic(err)
	}
	return newBot
}

func GetNotificationChannelID() string {
	channelId := getEnv("TRAQ_NOTIFICATION_CHANNEL_ID", "")
	if channelId == "" {
		panic("TRAQ_CHANNEL_ID is not set")
	}
	return channelId
}
