create table eval_batch_tasks (
    id serial primary key,
    dataset_id integer not null,
    "name" varchar(128) not null,
    task_uuid varchar(128) not null,
    user_id integer not null,
    status varchar(128) null, -- pending, completed, failed, in-process
    task_type integer not null,
    candidate_system_prompt text null,
    candidate_user_prompt text null,
    judge_system_prompt text null,
    judge_user_prompt text null,
    model_config_A jsonb null,
    model_config_B jsonb null,
    model_config_judge jsonb null,
    dataset_item integer not null,
    created_at timestamp with time zone default current_timestamp,
    updated_at timestamp with time zone default current_timestamp,
    created_by integer null,
    updated_by integer null,
    is_deleted boolean default false
)
create index idx_task_uuid_dataset_id on eval_batch_tasks(task_uuid, dataset_id)

create table eval_task_results (
    id serial primary key ,
    dataset_item_id integer not null ,
    task_uuid integer not null ,
    dataset_id integer not null,
    user_id integer not null,
    system_prompt_A text null,
    user_prompt_A text null,
    system_prompt_B text null,
    user_prompt_B text null,
    response_A text null,
    response_B text null,
    judge_response text null,
    task_type integer not null,
    created_at timestamp with time zone default current_timestamp,
    updated_at timestamp with time zone default current_timestamp,
    created_by integer null,
    updated_by integer null,
    is_deleted boolean default false
)
create index idx_task_uuid_dataset_id on eval_task_results(task_uuid, dataset_id)