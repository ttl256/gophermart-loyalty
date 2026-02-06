-- name: InsertUser :one
insert into users (id, login, password_hash)
values ($1, $2, $3)
returning id;

-- name: SelectUserByLogin :one
select id, login, password_hash, created_at
from users
where login = $1;
