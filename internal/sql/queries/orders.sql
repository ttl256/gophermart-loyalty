-- name: InsertOrder :one
insert into orders (number, user_id)
values ($1, $2)
on conflict(number) do nothing
returning user_id;

-- name: GetOrderOwner :one
select user_id
from orders
where number = $1;
