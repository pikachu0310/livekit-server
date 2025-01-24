package handler

import (
	"github.com/gorilla/websocket"
	"github.com/pikachu0310/livekit-server/internal/repository"
	"sync"
)

type Handler struct {
	repo    *repository.Repository
	Clients map[*websocket.Conn]bool
	Mutex   sync.Mutex
}

func New(repo *repository.Repository) *Handler {
	return &Handler{
		repo:    repo,
		Clients: make(map[*websocket.Conn]bool),
	}
}
