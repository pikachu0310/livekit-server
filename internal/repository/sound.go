package repository

import (
	"fmt"
)

// Sound は DB上の sounds テーブルに対応する構造体です
type Sound struct {
	SoundID   string `db:"sound_id"`   // UUIDを文字列で扱う
	SoundName string `db:"sound_name"` // サウンド名
	StampID   string `db:"stamp_id"`   // スタンプID（任意）
	CreatorID string `db:"creator_id"` // 作成者のID
}

// InsertSoundboardItem は (soundId, soundName, stampId, creatorId) を sounds テーブルへ登録します
func (r *Repository) InsertSoundboardItem(soundID, soundName, stampID, creatorID string) error {
	_, err := r.db.Exec(`
		INSERT INTO sounds (sound_id, sound_name, stamp_id, creator_id)
		VALUES (?, ?, ?, ?)
	`, soundID, soundName, stampID, creatorID)
	if err != nil {
		return fmt.Errorf("insert soundboard item: %w", err)
	}
	return nil
}

// GetAllSoundboards は sounds テーブルのレコードを全て取得します
func (r *Repository) GetAllSoundboards() ([]Sound, error) {
	var sounds []Sound
	if err := r.db.Select(&sounds, `
		SELECT sound_id, sound_name, stamp_id, creator_id
		FROM sounds
	`); err != nil {
		return nil, fmt.Errorf("select sounds: %w", err)
	}
	return sounds, nil
}

// GetSoundboardByCreatorID は指定された creator_id に関連する sounds を取得します
func (r *Repository) GetSoundboardByCreatorID(creatorID string) ([]Sound, error) {
	var sounds []Sound
	if err := r.db.Select(&sounds, `
		SELECT sound_id, sound_name, stamp_id, creator_id
		FROM sounds
		WHERE creator_id = ?
	`, creatorID); err != nil {
		return nil, fmt.Errorf("select sounds by creator_id: %w", err)
	}
	return sounds, nil
}

// EditSoundboardCreatorID は指定された sound_id の creator_id を更新します
func (r *Repository) EditSoundboardCreatorID(soundID, creatorID string) error {
	_, err := r.db.Exec(`
		UPDATE sounds
		SET creator_id = ?
		WHERE sound_id = ?
	`, creatorID, soundID)
	if err != nil {
		return fmt.Errorf("edit soundboard creator_id: %w", err)
	}
	return nil
}

// DeleteSoundboardItem は指定された sound_id のレコードを削除します
func (r *Repository) DeleteSoundboardItem(soundID string) error {
	_, err := r.db.Exec(`
		DELETE FROM sounds
		WHERE sound_id = ?
	`, soundID)
	if err != nil {
		return fmt.Errorf("delete soundboard item: %w", err)
	}
	return nil
}
