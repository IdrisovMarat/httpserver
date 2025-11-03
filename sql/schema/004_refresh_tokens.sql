
-- +goose Up
CREATE TABLE refresh_tokens (
    -- Primary key: токен как строка (256-bit hex)
    token TEXT PRIMARY KEY,
    
    -- Таймстампы для аудита и отслеживания
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Foreign key с каскадным удалением
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Время истечения токена
    expires_at TIMESTAMP NOT NULL,
    
    -- Время отзыва токена (NULL если активен)
    revoked_at TIMESTAMP,
    
    -- Production рекомендация: Индексы для производительности
    CONSTRAINT refresh_tokens_user_id_idx FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Production рекомендация: Создаем индексы для частых запросов
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX idx_refresh_tokens_revoked_at ON refresh_tokens(revoked_at);

-- Production рекомендация: Комментарии для документации БД
COMMENT ON TABLE refresh_tokens IS 'Таблица для хранения refresh tokens с возможностью отзыва';
COMMENT ON COLUMN refresh_tokens.token IS '256-bit hex encoded refresh token (primary key)';
COMMENT ON COLUMN refresh_tokens.revoked_at IS 'Timestamp отзыва токена (NULL если активен)';


-- +goose Down
DROP TABLE refresh_tokens;