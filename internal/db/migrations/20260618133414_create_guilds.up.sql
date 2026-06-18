CREATE TABLE guilds (
    id            UUID        PRIMARY KEY DEFAULT uuidv7(),
    name          TEXT        NOT NULL UNIQUE,
    gold_balance  BIGINT      NOT NULL DEFAULT 0,
    daily_limit   BIGINT      NOT NULL DEFAULT 10000,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- balance should never go negative
    CONSTRAINT positive_balance CHECK (gold_balance >= 0),
    CONSTRAINT positive_daily_limit CHECK (daily_limit > 0)
);