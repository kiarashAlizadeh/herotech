CREATE TYPE item_type AS ENUM ('common', 'rare', 'legendary');
CREATE TYPE item_status AS ENUM ('available', 'in_auction', 'sold');

CREATE TABLE items (
    id          UUID        PRIMARY KEY DEFAULT uuidv7(),
    name        TEXT        NOT NULL,
    type        item_type   NOT NULL,
    status      item_status NOT NULL DEFAULT 'available',
    owner_id    UUID        NOT NULL REFERENCES guilds(id),

    -- base_price comes from the price oracle at listing time
    base_price  BIGINT      NOT NULL,

    -- list_price is only set for common/rare items (limit orders)
    -- legendary items go through auction, so this stays null
    list_price  BIGINT,

    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT positive_base_price CHECK (base_price > 0),
    CONSTRAINT positive_list_price CHECK (list_price IS NULL OR list_price > 0)
);

-- legendary items are one-of-a-kind, enforce uniqueness by name
CREATE UNIQUE INDEX idx_legendary_unique_name
    ON items(name)
    WHERE type = 'legendary';

CREATE INDEX idx_items_owner ON items(owner_id);
CREATE INDEX idx_items_status ON items(status);