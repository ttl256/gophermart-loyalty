-- name: InsertOrder :one
insert into orders (number, user_id)
values ($1, $2)
on conflict(number) do nothing
returning user_id;

-- name: GetOrderOwner :one
select user_id
from orders
where number = $1;

-- name: GetOrders :many
select number, status, accrual, uploaded_at
from orders
where user_id = $1
order by uploaded_at desc;
