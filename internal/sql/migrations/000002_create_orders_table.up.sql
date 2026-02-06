create table if not exists orders (
    number text primary key,
    user_id uuid not null references users(id),
    status text not null default 'NEW',
    accrual numeric(12, 2) not null default 0,
    uploaded_at timestamptz not null default now()
);
