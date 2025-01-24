package handler

import (
	"encoding/json"
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/webhook"
	"net/http"
)

// LiveKitWebhook POST /webhook
func (h *Handler) LiveKitWebhook(c echo.Context) error {
	// Content-Type が application/webhook+json であることを許容
	if c.Request().Header.Get("Content-Type") != "application/webhook+json" {
		return c.JSON(http.StatusUnsupportedMediaType, map[string]string{
			"error": "Unsupported Content-Type",
		})
	}

	// リクエストボディをパース
	var event livekit.WebhookEvent
	if err := json.NewDecoder(c.Request().Body).Decode(&event); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid Webhook payload",
		})
	}

	// ルーム状態を更新
	switch event.Event {
	case webhook.EventParticipantJoined:
		h.repo.AddParticipantToRoomState(event.Room, event.Participant)
	case webhook.EventParticipantLeft:
		h.repo.RemoveParticipant(event.Room.Name, event.Participant.Identity)
	}

	// 全ルームの状態をWebSocketでブロードキャスト
	h.broadcastRoomState()

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
