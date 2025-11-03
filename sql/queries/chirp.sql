-- name: CreateChirp :one
INSERT INTO chirps (body, user_id)
VALUES ($1, $2)
RETURNING *;

-- name: DeleteAllChirps :exec
DELETE FROM chirps;

-- name: GetChirps :many
SELECT * FROM chirps
ORDER BY created_at ASC;

-- name: GetChirpsById :one
SELECT * FROM chirps
WHERE id = $1;


-- name: DeleteChirp :exec
DELETE FROM chirps 
WHERE id = $1;

-- name: GetChirpByID :one
SELECT * FROM chirps 
WHERE id = $1;

-- name: GetChirpsByAuthorID :many
SELECT * FROM chirps 
WHERE user_id = $1
ORDER BY created_at ASC;