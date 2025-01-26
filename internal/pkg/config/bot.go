package config

import (
	"fmt"
	traqwsbot "github.com/traPtitech/traq-ws-bot"
)

func NewTraQBot() *traqwsbot.Bot {
	newBot, err := traqwsbot.NewBot(&traqwsbot.Options{
		AccessToken: getEnv("TRAQ_ACCESS_TOKEN", ""),
		Origin:      getEnv("TRAQ_ORIGIN", "wss://q.trap.jp"),
	})
	if err != nil {
		fmt.Println("Failed to create new bot: " + err.Error())
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
