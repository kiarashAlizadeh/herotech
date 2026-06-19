-- tracks how much each guild has spent per day
-- used to enforce the daily purchase limit
CREATE TABLE daily_purchases (
    guild_id     UUID  NOT NULL REFERENCES guilds(id),
    date         DATE  NOT NULL DEFAULT CURRENT_DATE,
    total_spent  BIGINT NOT NULL DEFAULT 0,

    PRIMARY KEY (guild_id, date),

    CONSTRAINT non_negative_spent CHECK (total_spent >= 0)
);