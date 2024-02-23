create extension if not exists citext;

create table if not exists users (
    id bigserial primary key,
    created_at timestamp(0) with time zone not null default now(),
    username text not null,
    email citext unique not null,
    password_hash bytea not null,
    phone text,
    version integer not null default 1
);

create table if not exists sessions (
  token text primary key,
  data bytea not null,
  expiry timestamptz not null
);

create index sessions_expiry_idx on sessions (expiry);

create table if not exists listings (
    id bigserial primary key,
    user_id bigint references users(id) not null,
    title text not null,
    description text not null,
    banner text not null,
    pictures text[],
    location text not null,
    property_type text not null check (property_type in ('apartment', 'land', 'house')),
    price int not null,
    created_at timestamp(0) with time zone not null default now()
);

create table if not exists pictures (
    listing_id bigint references listings(id) not null,
    url text not null,
    created_at timestamptz(0) not null default now(),
    deleted_at timestamptz(0)
);

create table if not exists favorites (
    user_id bigint references users(id) not null,
    listing_id bigint references listings(id) not null,
    created_at timestamp(0) with time zone not null default now()
);

alter table favorites add constraint unique_user_listing_pair unique(user_id, listing_id);
