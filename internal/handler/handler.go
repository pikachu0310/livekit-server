package handler

import (
	"github.com/gorilla/websocket"
	"github.com/pikachu0310/livekit-server/internal/repository"
	"sync"
)

type Handler struct {
	repo        *repository.Repository
	Clients     map[*websocket.Conn]bool
	Mutex       sync.Mutex
	FileService *repository.FileService
}

func New(repo *repository.Repository, f *repository.FileService) *Handler {
	return &Handler{
		repo:        repo,
		Clients:     make(map[*websocket.Conn]bool),
		FileService: f,
	}
}
