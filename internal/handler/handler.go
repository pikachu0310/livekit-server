package handler

import (
	"github.com/gorilla/websocket"
	"github.com/pikachu0310/livekit-server/internal/pkg/config"
	"github.com/pikachu0310/livekit-server/internal/repository"
	"sync"
)

type Handler struct {
	repo        *repository.Repository
	LiveKitHost string
	ApiKey      string
	ApiSecret   string
	Clients     map[*websocket.Conn]bool
	Mutex       sync.Mutex
	RoomState   map[string][]string
}

func New(repo *repository.Repository, cfg *config.LivekitConfig) *Handler {
	return &Handler{
		repo:        repo,
		LiveKitHost: cfg.LiveKitHost,
		ApiKey:      cfg.ApiKey,
		ApiSecret:   cfg.ApiSecret,
		Clients:     make(map[*websocket.Conn]bool),
		RoomState:   make(map[string][]string),
	}
}
