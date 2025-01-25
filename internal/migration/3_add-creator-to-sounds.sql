-- +goose Up
ALTER TABLE sounds
    ADD COLUMN creator_id VARCHAR(36) NOT NULL;

-- +goose Down
ALTER TABLE sounds
    DROP COLUMN creator_id;
