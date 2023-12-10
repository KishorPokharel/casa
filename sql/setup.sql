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

create table if not exists listings (
    id bigserial primary key,
    created_at timestamp(0) with time zone not null default now(),
    description text not null,
    banner text not null,
    pictures text[],
    location text not null,
    property_type text not null check (property_type in ('apartment', 'land', 'house')),
    price int not null
);
