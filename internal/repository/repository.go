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

func New(db *sqlx.DB, liveKitCfg *config.LivekitConfig) *Repository {
	return &Repository{
		db:          db,
		LiveKitHost: liveKitCfg.LiveKitHost,
		ApiKey:      liveKitCfg.ApiKey,
		ApiSecret:   liveKitCfg.ApiSecret,
		RoomState:   make([]models.RoomWithParticipants, 0),
	}
}
