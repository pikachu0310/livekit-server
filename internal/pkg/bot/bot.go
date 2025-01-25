package bot

import (
	"context"
	"fmt"
	"github.com/pikachu0310/livekit-server/internal/pkg/config"
	"github.com/traPtitech/go-traq"
	traqwsbot "github.com/traPtitech/traq-ws-bot"
)

var (
	bot                   *traqwsbot.Bot
	notificationChannelId string
)

func SetAndStartTraQBot() {
	setNewTraQBot()
	setChannelID()
	startBotOnBackground()
}

func setNewTraQBot() {
	bot = config.NewTraQBot()
}

func startBotOnBackground() {
	go func() {
		if err := bot.Start(); err != nil {
			panic(err)
		}
	}()
}

func setChannelID() {
	notificationChannelId = config.GetNotificationChannelID()
}

func SendMessageToNotificationChannel(content string) {
	_, err := SendMessage(notificationChannelId, content, true)
	if err != nil {
		fmt.Println("Failed to send message to notification channel: " + err.Error())
	}
}

func SendMessage(channelID, content string, embed bool) (*traq.Message, error) {
	message, _, err := bot.API().
		MessageApi.
		PostMessage(context.Background(), channelID).
		PostMessageRequest(traq.PostMessageRequest{
			Content: content,
			Embed:   &embed,
		}).
		Execute()
	return message, err
}

func GetChannels() (*traq.ChannelList, error) {
	channels, _, err := bot.API().
		ChannelApi.
		GetChannels(context.Background()).
		Execute()
	if err != nil {
		return nil, err
	}

	return channels, err
}

func GetChannel(channelID string) (*traq.Channel, error) {
	channel, _, err := bot.API().
		ChannelApi.
		GetChannel(context.Background(), channelID).
		Execute()
	if err != nil {
		return nil, err
	}

	return channel, err
}

func GetChannelPath(channelID string) (string, error) {
	path, _, err := bot.API().
		ChannelApi.
		GetChannelPath(context.Background(), channelID).
		Execute()
	if err != nil {
		return "", err
	}

	return path.Path, err
}

func GetStamp(stampID string) (*traq.Stamp, error) {
	stamp, _, err := bot.API().
		StampApi.
		GetStamp(context.Background(), stampID).
		Execute()
	if err != nil {
		return nil, err
	}

	return stamp, err
}

func GetUser(userID string) (*traq.UserDetail, error) {
	user, _, err := bot.API().
		UserApi.
		GetUser(context.Background(), userID).
		Execute()
	if err != nil {
		return nil, err
	}

	return user, err
}

func GetUserByName(userName string) (*traq.User, error) {
	user, _, err := bot.API().
		UserApi.
		GetUsers(context.Background()).
		Name(userName).
		Execute()
	if err != nil {
		return nil, err
	}

	if len(user) == 1 {
		return &user[0], err
	}

	return nil, fmt.Errorf("user not found")
}

/*
func main() {
	bot.OnMessageCreated(func(p *payload.MessageCreated) {
		log.Println("Received MESSAGE_CREATED event: " + p.Message.Text)
		_, _, err := bot.API().
			MessageApi.
			PostMessage(context.Background(), p.Message.ChannelID).
			PostMessageRequest(traq.PostMessageRequest{
				Content: "oisu-",
			}).
			Execute()
		if err != nil {
			log.Println(err)
		}
	})

	if err := bot.Start(); err != nil {
		panic(err)
	}
}
*/
