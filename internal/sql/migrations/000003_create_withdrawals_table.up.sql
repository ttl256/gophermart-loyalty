create table if not exists withdrawals (
    id uuid primary key,
    user_id uuid not null references users(id),
    order_number text not null,
    sum numeric(12, 2) not null check (sum > 0),
    processed_at timestamptz not null default now()
);
