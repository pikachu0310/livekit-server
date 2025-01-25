-- +goose Up
CREATE TABLE IF NOT EXISTS sounds
(
    sound_id   VARCHAR(36)  NOT NULL,
    sound_name VARCHAR(255) NOT NULL,
    stamp_id   VARCHAR(255),
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (sound_id)
);

-- +goose Down
DROP TABLE IF EXISTS sounds;
