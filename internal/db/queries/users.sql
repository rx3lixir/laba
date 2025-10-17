-- name: CreateUser :one
INSERT INTO users (id, name, email, password)
VALUES (gen_random_uuid(), $1, $2, $3)
RETURNING id, name, email, password, created_at, updated_at;

-- name: GetUserByID :one
select id, name, email, password, created_at, updated_at
from users
where id = $1
;

-- name: GetUserByEmail :one
select id, name, email, password, created_at, updated_at
from users
where email = $1
;

-- name: ListUsers :many
select id, name, email, created_at, updated_at
from users
order by created_at desc
limit $1
offset $2
;

-- name: UpdateUser :one
UPDATE users
SET 
  name = COALESCE($2, name),
  email = COALESCE($3, email),
  updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, name, email, created_at, updated_at;

-- name: DeleteUser :exec
delete from users
where id = $1
;

-- name: CountUsers :one
select count(*)
from users
;

