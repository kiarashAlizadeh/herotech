CREATE TABLE bids (
    id          UUID        PRIMARY KEY DEFAULT uuidv7(),
    auction_id  UUID        NOT NULL REFERENCES auctions(id),
    bidder_id   UUID        NOT NULL REFERENCES guilds(id),
    amount      BIGINT      NOT NULL,

    -- when a bid is outbid or cancelled, it becomes inactive
    -- and the reserved gold is released back to the guild
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,

    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT positive_bid_amount CHECK (amount > 0)
);

-- each guild can only have one active bid per auction
CREATE UNIQUE INDEX idx_one_active_bid_per_bidder
    ON bids(auction_id, bidder_id)
    WHERE is_active = TRUE;

CREATE INDEX idx_bids_auction ON bids(auction_id);
CREATE INDEX idx_bids_bidder ON bids(bidder_id);