package handler

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/webhook"
	"net/http"
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
		//h.repo.AddParticipantToRoomState(event.Room, event.Participant)
		err := h.repo.GetRoomsWithParticipantsByLiveKitServerAndSave(c.Request().Context())
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error on GetRoomsWithParticipantsByLiveKitServer": err.Error()})
		}
	case webhook.EventParticipantLeft:
		fmt.Printf("Participant left: room=%s, participant=%s", event.Room.Name, event.Participant.Identity)
		//h.repo.RemoveParticipant(event.Room.Name, event.Participant.Identity)
		err := h.repo.GetRoomsWithParticipantsByLiveKitServerAndSave(c.Request().Context())
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error on GetRoomsWithParticipantsByLiveKitServer": err.Error()})
		}
	default:
		fmt.Printf("Unhandled webhook event: %s", event.Event)
		err := h.repo.GetRoomsWithParticipantsByLiveKitServerAndSave(c.Request().Context())
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error on GetRoomsWithParticipantsByLiveKitServer": err.Error()})
		}
	}

	// 全ルームの状態をWebSocketでブロードキャスト
	h.broadcastRoomState()

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
