-- +goose Up

-- Add performance indexes for better query performance

-- Holdings table indexes for client_id + date range queries
-- These are the most common query patterns in holdings repository
CREATE INDEX IF NOT EXISTS idx_holdings_client_date_range ON holdings(client_id, date);
CREATE INDEX IF NOT EXISTS idx_holdings_date_client ON holdings(date, client_id);

-- Transactions table indexes for client_id + date range queries
-- These are the most common query patterns in transactions repository
CREATE INDEX IF NOT EXISTS idx_transactions_client_date_range ON transactions(client_id, date);
CREATE INDEX IF NOT EXISTS idx_transactions_date_client ON transactions(date, client_id);

-- Composite indexes for JOIN operations in holdings
-- Optimizes queries that join holdings with assets and categories
CREATE INDEX IF NOT EXISTS idx_holdings_asset_date ON holdings(asset_id, date);
CREATE INDEX IF NOT EXISTS idx_holdings_client_asset_date ON holdings(client_id, asset_id, date);

-- Composite indexes for JOIN operations in transactions
-- Optimizes queries that join transactions with assets and categories
CREATE INDEX IF NOT EXISTS idx_transactions_asset_date ON transactions(asset_id, date);
CREATE INDEX IF NOT EXISTS idx_transactions_client_asset_date ON transactions(client_id, asset_id, date);

-- Indexes for aggregation queries (GROUP BY operations)
-- Optimizes queries that group by date and category
CREATE INDEX IF NOT EXISTS idx_holdings_date_for_aggregation ON holdings(date) WHERE deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_transactions_date_for_aggregation ON transactions(date) WHERE deleted = FALSE;

-- Partial indexes for soft delete filtering
-- These indexes only include non-deleted records, improving performance for active data queries
CREATE INDEX IF NOT EXISTS idx_holdings_active_client_date ON holdings(client_id, date) WHERE deleted = FALSE;
CREATE INDEX IF NOT EXISTS idx_transactions_active_client_date ON transactions(client_id, date) WHERE deleted = FALSE;

-- Indexes for sync_logs date range queries (optimization)
-- The existing composite index should be sufficient, but adding a date-only index for broader queries
CREATE INDEX IF NOT EXISTS idx_sync_logs_date ON sync_logs(sync_date);

-- Indexes for asset lookups by external_id (optimization)
-- The unique constraint already provides an index, but adding a partial index for non-deleted assets
CREATE INDEX IF NOT EXISTS idx_assets_active_external_id ON assets(external_id) WHERE deleted = FALSE;

-- Indexes for asset category lookups (optimization)
-- Adding partial index for non-deleted categories
CREATE INDEX IF NOT EXISTS idx_asset_categories_active_name ON asset_categories(name) WHERE deleted = FALSE;

-- +goose Down

-- Drop all the indexes we created
DROP INDEX IF EXISTS idx_holdings_client_date_range;
DROP INDEX IF EXISTS idx_holdings_date_client;
DROP INDEX IF EXISTS idx_transactions_client_date_range;
DROP INDEX IF EXISTS idx_transactions_date_client;
DROP INDEX IF EXISTS idx_holdings_asset_date;
DROP INDEX IF EXISTS idx_holdings_client_asset_date;
DROP INDEX IF EXISTS idx_transactions_asset_date;
DROP INDEX IF EXISTS idx_transactions_client_asset_date;
DROP INDEX IF EXISTS idx_holdings_date_for_aggregation;
DROP INDEX IF EXISTS idx_transactions_date_for_aggregation;
DROP INDEX IF EXISTS idx_holdings_active_client_date;
DROP INDEX IF EXISTS idx_transactions_active_client_date;
DROP INDEX IF EXISTS idx_sync_logs_date;
DROP INDEX IF EXISTS idx_assets_active_external_id;
DROP INDEX IF EXISTS idx_asset_categories_active_name;
