package repository

import (
	"fmt"
	"github.com/pikachu0310/livekit-server/internal/pkg/bot"
	"github.com/traPtitech/go-traq"
)

var (
	traQChannels map[string]*traq.Channel
)

func (r *Repository) SendJoinMessageToTraQ(channelId string, userName string) {
	path := r.GetChannelFullPath(channelId)
	content := fmt.Sprintf(":@%s: %s さんが #%s に参加しました", userName, userName, path)
	bot.SendMessageToNotificationChannel(content)
}

func (r *Repository) SendLeaveMessageToTraQ(channelId string, userName string) {
	path := r.GetChannelFullPath(channelId)
	content := fmt.Sprintf(":@%s: %s さんが #%s から退出しました", userName, userName, path)
	bot.SendMessageToNotificationChannel(content)
}

func (r *Repository) SendStartRoomMessageToTraQ(channelId string) {
	path := r.GetChannelFullPath(channelId)
	content := fmt.Sprintf("#%s で Qall が開始されました", path)
	bot.SendMessageToNotificationChannel(content)
}

func (r *Repository) SendEndRoomMessageToTraQ(channelId string) {
	path := r.GetChannelFullPath(channelId)
	content := fmt.Sprintf("#%s で Qall が終了しました", path)
	bot.SendMessageToNotificationChannel(content)
}

func (r *Repository) SendStartScreenShareMessageToTraQ(channelId string, userName string) {
	path := r.GetChannelFullPath(channelId)
	content := fmt.Sprintf(":@%s: %s さんが #%s で画面共有を開始しました", userName, userName, path)
	bot.SendMessageToNotificationChannel(content)
}

func (r *Repository) GetTraQChannelsAndSet() error {
	channels, err := bot.GetChannels()
	if err != nil {
		return err
	}

	traQChannels = make(map[string]*traq.Channel)
	for _, channel := range channels.Public {
		traQChannels[channel.Id] = &channel
	}
	return nil
}

func (r *Repository) CheckChannelExistence(channelId string) bool {
	_, ok := traQChannels[channelId]
	if !ok {
		err := r.GetTraQChannelsAndSet()
		if err != nil {
			return false
		}
		_, ok = traQChannels[channelId]
	}
	return ok
}

func (r *Repository) GetChannelFullPath(channelId string) string {
	path, err := bot.GetChannelPath(channelId)
	if err != nil {
		return ""
	}

	if path[0] == '/' && len(path) > 1 {
		return path[1:]
	}

	return path
}

func (r *Repository) CheckStampExistence(stampId string) bool {
	_, err := bot.GetStamp(stampId)
	return err == nil
}

func (r *Repository) CheckUserExistence(userId string) bool {
	_, err := bot.GetUser(userId)
	return err == nil
}

func (r *Repository) CheckUserExistenceByName(userName string) bool {
	_, err := bot.GetUserByName(userName)
	return err == nil
}
