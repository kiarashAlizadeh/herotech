CREATE TYPE auction_status AS ENUM ('active', 'ended', 'cancelled');

CREATE TABLE auctions (
    id           UUID           PRIMARY KEY DEFAULT uuidv7(),
    item_id      UUID           NOT NULL REFERENCES items(id),
    seller_id    UUID           NOT NULL REFERENCES guilds(id),
    status       auction_status NOT NULL DEFAULT 'active',

    start_price  BIGINT         NOT NULL,
    highest_bid  BIGINT,
    winner_id    UUID           REFERENCES guilds(id),

    ends_at      TIMESTAMPTZ    NOT NULL,
    created_at   TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT positive_start_price CHECK (start_price > 0),
    CONSTRAINT positive_highest_bid CHECK (highest_bid IS NULL OR highest_bid > 0)
);

-- a legendary item can only have one active auction at a time
CREATE UNIQUE INDEX idx_one_active_auction_per_item
    ON auctions(item_id)
    WHERE status = 'active';

CREATE INDEX idx_auctions_status ON auctions(status);
CREATE INDEX idx_auctions_ends_at ON auctions(ends_at) WHERE status = 'active';