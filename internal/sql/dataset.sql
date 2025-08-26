create table if not exists dataset_metadata (
    id serial primary key,
    `name` text not null,
    description text,
    created_at timestamp with time zone default current_timestamp,
    updated_at timestamp with time zone default current_timestamp,
    
);