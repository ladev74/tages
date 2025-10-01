create schema if not exists schema_files;

create table if not exists schema_files.table_files
(
    id uuid primary key,
    name text not null ,
    created_at timestamptz not null ,
    updated_at timestamptz not null ,
    status text not null
);