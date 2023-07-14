BEGIN;

CREATE TYPE CATEGORY AS ENUM ('0', '1');

CREATE TABLE IF NOT EXISTS groups
(
    id             SMALLSERIAL PRIMARY KEY,
    name           VARCHAR(256),
    description    VARCHAR(1024),
    tag            VARCHAR(256),
    link           VARCHAR(256),
    stopwords      VARCHAR(1024),
    n_days         INT,
    scan_time      TIMESTAMP WITH TIME ZONE,
    last_scan_time TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS sources
(
    id             SERIAL PRIMARY KEY,
    category       CATEGORY,
    link           VARCHAR(256),
    duration_limit INT,
    like_limit     INT,
    comment_limit  INT,
    repost_limit   INT,
    view_limit     INT,
    group_id       INT REFERENCES groups (id)
);

CREATE TABLE IF NOT EXISTS events
(
    id              BIGSERIAL PRIMARY KEY,
    category        CATEGORY,
    date_time       TIMESTAMP WITH TIME ZONE,
    repeat_interval INT,
    group_id        INT REFERENCES groups (id)
);

COMMIT;