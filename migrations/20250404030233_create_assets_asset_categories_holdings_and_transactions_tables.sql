-- +goose Up

-- Create asset_categories table
CREATE TABLE asset_categories (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP
);

-- Create assets table
CREATE TABLE assets (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    external_id TEXT NOT NULL UNIQUE,
    name TEXT,
    asset_type TEXT,
    category_id INTEGER REFERENCES asset_categories(id) ON DELETE CASCADE,
    currency TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_assets_category_id ON assets(category_id);

-- Create holdings table
CREATE TABLE holdings (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    client_id TEXT NOT NULL,
    asset_id INTEGER REFERENCES assets(id) ON DELETE CASCADE,
    quantity NUMERIC NOT NULL,
    value NUMERIC,
    date DATE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_holdings_client_id ON holdings(client_id);
CREATE INDEX idx_holdings_asset_id ON holdings(asset_id);
CREATE INDEX idx_holdings_date ON holdings(date);

-- Create transactions table
CREATE TABLE transactions (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    client_id TEXT NOT NULL,
    asset_id INTEGER REFERENCES assets(id) ON DELETE CASCADE,
    transaction_type TEXT,
    quantity NUMERIC NOT NULL,
    price_per_unit NUMERIC,
    total_value NUMERIC,
    date DATE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN DEFAULT FALSE,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_transactions_client_id ON transactions(client_id);
CREATE INDEX idx_transactions_asset_id ON transactions(asset_id);
CREATE INDEX idx_transactions_date ON transactions(date);

-- Create sync_logs table
CREATE TABLE sync_logs (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    client_id TEXT NOT NULL,
    sync_date DATE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sync_logs_client_date ON sync_logs(client_id, sync_date);

-- +goose Down
DROP TABLE IF EXISTS sync_logs;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS holdings;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS asset_categories;
