package repository

import (
	"fmt"
)

// Sound は DB上の sounds テーブルに対応する構造体です
type Sound struct {
	SoundID   string `db:"sound_id"` // UUIDを文字列で扱う
	SoundName string `db:"sound_name"`
	StampID   string `db:"stamp_id"`
}

// InsertSoundboardItem は (soundId, soundName, stampId) を sounds テーブルへ登録します
func (r *Repository) InsertSoundboardItem(soundID, soundName, stampID string) error {
	_, err := r.db.Exec(`
		INSERT INTO sounds (sound_id, sound_name, stamp_id)
		VALUES (?, ?, ?)
	`, soundID, soundName, stampID)
	if err != nil {
		return fmt.Errorf("insert soundboard item: %w", err)
	}
	return nil
}

// GetAllSoundboards は sounds テーブルのレコードを全て取得します
func (r *Repository) GetAllSoundboards() ([]Sound, error) {
	var sounds []Sound
	if err := r.db.Select(&sounds, `
		SELECT sound_id, sound_name, stamp_id
		FROM sounds
	`); err != nil {
		return nil, fmt.Errorf("select sounds: %w", err)
	}
	return sounds, nil
}
