-- name: GetOrdersForProcessing :many
select number, user_id, status, accrual, uploaded_at
from orders
where status in ('NEW','PROCESSING')
order by uploaded_at asc;

-- name: UpdateOrderStatus :exec
update orders
set status = $2,
    accrual = $3
where number = $1;
