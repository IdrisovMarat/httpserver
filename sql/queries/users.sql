-- name: CreateUser :one
INSERT INTO users (email, hashed_password)
VALUES ($1, $2)
RETURNING *;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: GetUserByEmail :one
SELECT * FROM users 
WHERE email = $1;

-- name: UpdateUser :one
UPDATE users 
SET email = $1, 
    hashed_password = $2,
    updated_at = NOW()
WHERE id = $3
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users 
WHERE id = $1;

-- name: UpgradeUserToChirpyRed :exec
UPDATE users 
SET is_chirpy_red = true,
    updated_at = NOW()
WHERE id = $1;
