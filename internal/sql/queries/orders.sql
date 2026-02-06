-- name: InsertOrder :one
insert into orders (number, user_id, status, accrual)
values ($1, $2, $3, $4)
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

-- name: GetBalance :one
with
accr as (
    select coalesce(sum(o.accrual), 0::numeric(12,2)) as total
    from orders o
    where o.user_id = $1
        and o.status = 'PROCESSED'
),
wd as (
    select coalesce(sum(w.sum), 0::numeric(12,2)) as total
    from withdrawals w
    where w.user_id = $1
)
select
    (accr.total - wd.total)::numeric(12,2) as current,
    wd.total::numeric(12,2) as withdrawn
from accr, wd;

-- name: InsertWithdrawal :exec
insert into withdrawals (user_id, order_number, sum)
values ($1, $2, $3);

-- name: AcquireUserLock :exec
select pg_advisory_xact_lock(hashtextextended(sqlc.arg(user_id)::uuid::text, 0));

-- name: GetWithdrawals :many
select order_number, sum, processed_at
from withdrawals
where user_id = $1
order by processed_at desc;
