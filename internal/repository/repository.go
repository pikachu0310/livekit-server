package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/pikachu0310/livekit-server/internal/pkg/config"
	"github.com/pikachu0310/livekit-server/openapi/models"
)

type Repository struct {
	db          *sqlx.DB
	LiveKitHost string
	ApiKey      string
	ApiSecret   string
	RoomState   []models.RoomWithParticipants
}

func New(db *sqlx.DB, cfg *config.LivekitConfig) *Repository {
	return &Repository{
		db:          db,
		LiveKitHost: cfg.LiveKitHost,
		ApiKey:      cfg.ApiKey,
		ApiSecret:   cfg.ApiSecret,
		RoomState:   make([]models.RoomWithParticipants, 0),
	}
}
