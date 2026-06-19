CREATE TYPE transaction_type AS ENUM (
    'deposit',   -- gold added to wallet
    'purchase',  -- limit order completed
    'reserve',   -- gold locked for an active bid
    'release',   -- reserved gold returned after being outbid or cancelling
    'refund'     -- seller receives gold after auction ends
);

CREATE TABLE wallet_transactions (
    id           UUID             PRIMARY KEY DEFAULT uuidv7(),
    guild_id     UUID             NOT NULL REFERENCES guilds(id),
    type         transaction_type NOT NULL,

    -- positive = gold in, negative = gold out
    amount       BIGINT           NOT NULL,

    -- points to the bid, auction, or item that caused this transaction
    reference_id UUID,

    description  TEXT,
    created_at   TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wallet_tx_guild ON wallet_transactions(guild_id);
CREATE INDEX idx_wallet_tx_created ON wallet_transactions(created_at);