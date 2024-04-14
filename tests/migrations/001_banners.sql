-- +goose up
CREATE TABLE IF NOT EXISTS banners (
    id SERIAL primary key,
    feature_id int,
    tag_ids int[],
    is_active boolean,
    updated_at timestamptz not null DEFAULT current_timestamp,
    created_at timestamptz not null DEFAULT current_timestamp,
    content text
);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL primary key,
    username varchar(32) unique,
    password_hash text,
    user_role varchar(32),
    feature_id int,
    tag_ids int[]   
);

INSERT INTO users(username, password_hash, user_role, feature_id, tag_ids)
VALUES
    ('Admin', '$2a$10$gAGkHH0MDiRi9LjyB3Xjdu02gUXmQ83ByWHWG4qYTygOC7hdsUFJi', 'admin', 1, ARRAY[1,2]),
    ('default_user', '$2a$10$PqeySSujKaxvhElDDQBGEud3XO1CGMZU2k8W7oeVlVhNPcB4RoYKq', 'user', 5, ARRAY[1,2,3,4]);

-- +goose down
DROP TABLE IF EXISTS banners;
DROP TABLE IF EXISTS users;