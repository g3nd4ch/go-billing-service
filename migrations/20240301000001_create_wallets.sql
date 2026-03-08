-- +goose Up
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY,
    balance NUMERIC NOT NULL DEFAULT 0,
    currency TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wallets_currency ON wallets (currency);

-- +goose Down
DROP TABLE IF EXISTS wallets;