create table if not exists dataset_metadata (
    id serial primary key,
    "name" text not null,
    description text,
    file_path text not null,
    created_at timestamp with time zone default current_timestamp,
    updated_at timestamp with time zone default current_timestamp,
    created_by integer null,
    updated_by integer null,
    is_deleted boolean default false
);
create index idx_id_name on dataset_metadata (id, "name");

create table if not exists dataset_item (
    id serial primary key,
    dataset_id integer not null,
    raw_content jsonb not null,
    user_id integer not null,
    created_at timestamp with time zone default current_timestamp,
    updated_at timestamp with time zone default current_timestamp,
    ctreated_by integer null,
    updated_by integer null,
    is_deleted boolean default false
);
create index idx_id_dataset_id on dataset_item (id, dataset_id);

create table if not exisits dataset_io_jobs (
    id serial primary key,
    dataset_id integer not null,
    job_type text not null,
    status text not null,
    created_at timestamp with time zone default current_timestamp,
    updated_at timestamp with time zone default current_timestamp,
    created_by integer null,
    updated_by integer null,
    is_deleted boolean default false
)
create index idx_id_dataset_id_job_type on dataset_io_jobs (id, dataset_id, job_type);