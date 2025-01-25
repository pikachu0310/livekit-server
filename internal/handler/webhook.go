package handler

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/webhook"
)

// LiveKitWebhook POST /webhook
func (h *Handler) LiveKitWebhook(c echo.Context) error {
	// Authプロバイダーを初期化
	authProvider := auth.NewSimpleKeyProvider(h.repo.ApiKey, h.repo.ApiSecret)

	// Webhookイベントを受け取る
	event, err := webhook.ReceiveWebhookEvent(c.Request(), authProvider)
	if err != nil {
		fmt.Printf("Failed to validate webhook: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Failed to validate webhook",
		})
	}

	// ルーム状態を更新
	switch event.Event {
	case webhook.EventParticipantJoined:
		fmt.Printf("Participant joined: room=%s, participant=%s", event.Room.Name, event.Participant.Identity)
		h.repo.AddParticipantToRoomState(event.Room, event.Participant)
		h.repo.SendJoinMessageToTraQ(event.Room.Name, event.Participant.Name)
	case webhook.EventParticipantLeft:
		fmt.Printf("Participant left: room=%s, participant=%s", event.Room.Name, event.Participant.Identity)
		h.repo.RemoveParticipant(event.Room.Name, event.Participant.Identity)
		h.repo.SendLeaveMessageToTraQ(event.Room.Name, event.Participant.Name)
	case webhook.EventRoomFinished:
		fmt.Printf("Room finished: room=%s", event.Room.Name)
		h.repo.RemoveRoomState(event.Room.Name)
		h.repo.SendEndRoomMessageToTraQ(event.Room.Name)
	case webhook.EventTrackPublished:
		fmt.Printf("Track published: room=%s, participant=%s, track=%s", event.Room.Name, event.Participant.Identity, event.Track.Sid)
		if h.repo.CheckUserExistenceByName(event.Participant.Name) {
			if len(event.Participant.Attributes) >= 1 {
				h.repo.SendStartScreenShareMessageToTraQ(event.Room.Name, event.Participant.Name)
			}
		}
	default:
		fmt.Printf("Unhandled webhook event: %s", event.Event)
	}

	// 全ルームの状態をWebSocketでブロードキャスト
	h.broadcastRoomState()

	return c.NoContent(http.StatusOK)
}
