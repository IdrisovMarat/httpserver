-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, user_id, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens 
WHERE token = $1;

-- name: GetUserFromRefreshToken :one
SELECT users.* FROM users
JOIN refresh_tokens ON users.id = refresh_tokens.user_id
WHERE refresh_tokens.token = $1 
  AND refresh_tokens.expires_at > NOW()
  AND refresh_tokens.revoked_at IS NULL;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens 
SET revoked_at = NOW(), updated_at = NOW()
WHERE token = $1;

-- Production рекомендация: Очистка истекших токенов (для maintenance)
-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens 
WHERE expires_at < NOW();

-- Production рекомендация: Отзыв всех токенов пользователя (при смене пароля и т.д.)
-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens 
SET revoked_at = NOW(), updated_at = NOW()
WHERE user_id = $1 AND revoked_at IS NULL;