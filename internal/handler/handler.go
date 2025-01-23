package handler

import (
	"github.com/pikachu0310/livekit-server/internal/pkg/config"
	"github.com/pikachu0310/livekit-server/internal/repository"
)

type Handler struct {
	repo        *repository.Repository
	LiveKitHost string
	ApiKey      string
	ApiSecret   string
}

func New(repo *repository.Repository, cfg *config.LivekitConfig) *Handler {
	return &Handler{
		repo:        repo,
		LiveKitHost: cfg.LiveKitHost,
		ApiKey:      cfg.ApiKey,
		ApiSecret:   cfg.ApiSecret,
	}
}
