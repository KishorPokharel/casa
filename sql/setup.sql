/* create table users if not exists ( */
/*     username text not null, */
/*     email citext not null, */
/*     password bytea not null, */
/*     phone text, */
/*     user_type {buyer, seller} */
/* ); */

create table if not exists property_listings (
    description text not null,
    banner text not null,
    pictures text[],
    location text not null,
    /* location_point geography not null, */
    property_type text not null check (property_type in ('apartment', 'land', 'house')),
    price int not null
);
