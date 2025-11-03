-- +goose Up
ALTER TABLE users 
ADD COLUMN is_chirpy_red BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN users.is_chirpy_red IS 'Флаг подписки на Chirpy Red';


-- +goose Down
ALTER TABLE users
DROP COLUMN is_chirpy_red;