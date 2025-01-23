package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (h *Handler) PingServer(c echo.Context) error {
	return c.String(http.StatusOK, "pong")
}

func (h *Handler) Test(_ echo.Context) error {
	//TODO implement me
	panic("implement me")
}
