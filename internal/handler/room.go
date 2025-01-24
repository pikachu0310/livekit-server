package handler

import (
	"github.com/labstack/echo/v4"
	"net/http"
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
