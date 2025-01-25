package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pikachu0310/livekit-server/internal/pkg/util"
	"github.com/pikachu0310/livekit-server/openapi/models"
)

// GetRooms GET /rooms
// 全ルームを取得し、それぞれの参加者一覧をまとめて返す。
func (h *Handler) GetRooms(ctx echo.Context) error {
	roomWithParticipants, err := h.repo.GetRoomsWithParticipantsByLiveKitServer(ctx.Request().Context())
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{
			"error on GetRoomsWithParticipantsByLiveKitServer": err.Error(),
		})
	}
	return ctx.JSON(http.StatusOK, roomWithParticipants)
}

// GetRoomMetadata GET /rooms/:room_id/metadata
// ルームのメタデータを取得する。
func (h *Handler) GetRoomMetadata(ctx echo.Context, roomID uuid.UUID) error {
	apiKey := h.repo.ApiKey
	apiSecret := h.repo.ApiSecret
	apiHost := h.repo.LiveKitHost
	if apiKey == "" || apiSecret == "" || apiHost == "" {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{
			"error": "API key, secret and host must be set in environment variables",
		})
	}

	_, echoErr := util.AuthTraQClient(ctx)
	if echoErr != nil {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{
			"error on AuthTraQClient": echoErr.Error(),
		})
	}

	for _, room := range h.repo.RoomState {
		if room.RoomId.String() == roomID.String() {
			return ctx.JSON(http.StatusOK, room.Metadata)
		}
	}

	return ctx.JSON(http.StatusNotFound, map[string]string{
		"error": "Room not found",
	})
}

// PatchRoomMetadata PATCH /rooms/:room_id/metadata
// ルームのメタデータを変更する。
func (h *Handler) ChangeRoomMetadata(ctx echo.Context, roomID uuid.UUID) error {
	// リクエストボディを取得
	type roomMetadataRequest struct {
		Metadata string `json:"metadata"`
	}
	var req roomMetadataRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"error on Bind": err.Error(),
		})
	}

	apiKey := h.repo.ApiKey
	apiSecret := h.repo.ApiSecret
	apiHost := h.repo.LiveKitHost
	if apiKey == "" || apiSecret == "" || apiHost == "" {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{
			"error": "API key, secret and host must be set in environment variables",
		})
	}

	c := lksdk.NewRoomServiceClient(apiHost, apiKey, apiSecret)

	userID, echoErr := util.AuthTraQClient(ctx)
	if echoErr != nil {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{
			"error on AuthTraQClient": echoErr.Error(),
		})
	}

	// ルームのメタデータを変更
	for _, room := range h.repo.RoomState {
		if room.RoomId.String() == roomID.String() {
			// ルームに参加しているか確認
			for _, participant := range room.Participants {
				if *participant.Identity == userID {
					// ルームのメタデータを変更
					room.Metadata = &req.Metadata
					metadata := &util.Metadata{
						Metadata:  req.Metadata,
						IsWebinar: *room.IsWebinar,
					}
					metadataStr, err := json.Marshal(metadata)
					if err != nil {
						return ctx.JSON(http.StatusInternalServerError, map[string]string{
							"error on Marshal": err.Error(),
						})
					}
					_, err = c.UpdateRoomMetadata(ctx.Request().Context(), &livekit.UpdateRoomMetadataRequest{
						Room:     roomID.String(),
						Metadata: string(metadataStr),
					})
					if err != nil {
						return ctx.JSON(http.StatusInternalServerError, map[string]string{
							"error on UpdateRoom": err.Error(),
						})
					}

					// 全体に通知
					h.broadcastRoomState()

					return ctx.JSON(http.StatusOK, map[string]string{})
				}
			}
			return ctx.JSON(http.StatusForbidden, map[string]string{
				"error": "You don't have permission to change room metadata",
			})
		}
	}

	return ctx.JSON(http.StatusNotFound, map[string]string{
		"error": "Room not found",
	})

}

// PatchRoomParticipants PATCH /rooms/:room_id/participants
// ルームの発言権限を変更する。
func (h *Handler) ChangeParticipantRole(ctx echo.Context, roomID uuid.UUID) error {
	// リクエストボディを取得
	var req []models.Participant
	var succeedUsers []string
	var failedUsers = map[string]string{}
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"error on Bind": err.Error(),
		})
	}

	apiKey := h.repo.ApiKey
	apiSecret := h.repo.ApiSecret
	apiHost := h.repo.LiveKitHost
	if apiKey == "" || apiSecret == "" || apiHost == "" {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{
			"error": "API key, secret and host must be set in environment variables",
		})
	}

	userID, err := util.AuthTraQClient(ctx)
	if err != nil {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{
			"error on AuthTraQClient": err.Error(),
		})
	}

	// ルームが存在するか確認
	for _, roomState := range h.repo.RoomState {
		if roomState.RoomId == roomID {
			// userがcanPublishかどうかを確認
			canPublish := false
			for _, participant := range roomState.Participants {
				if *participant.Identity == userID {
					canPublish = *participant.CanPublish
					break
				}
			}
			if !canPublish {
				return ctx.JSON(http.StatusForbidden, map[string]string{
					"error": "You don't have permission to change participant role",
				})
			}
			c := lksdk.NewRoomServiceClient(apiHost, apiKey, apiSecret)
			for _, participant := range req {
				_, err := c.UpdateParticipant(ctx.Request().Context(), &livekit.UpdateParticipantRequest{
					Room:     roomID.String(),
					Identity: *participant.Identity,
					Permission: &livekit.ParticipantPermission{
						CanPublish: *participant.CanPublish,
					},
				})
				if err != nil {
					failedUsers[*participant.Identity] = err.Error()
				} else {
					succeedUsers = append(succeedUsers, *participant.Identity)
					h.repo.UpdateParticipantCanPublish(roomID.String(), *participant.Identity, *participant.CanPublish)
				}

			}
		}
		break
	}

	response := make([]map[string]string, 0)

	// Add successful participants
	for _, id := range succeedUsers {
		response = append(response, map[string]string{
			"participantId": id,
			"status":        "success",
		})
	}

	// Add failed participants
	for _, id := range failedUsers {
		response = append(response, map[string]string{
			"participantId": id,
			"status":        "failed",
			"error":         failedUsers[id],
		})
	}

	// 全体に通知
	h.broadcastRoomState()

	return ctx.JSON(http.StatusOK, response)

}
