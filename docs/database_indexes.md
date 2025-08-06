# Database Indexes Documentation

This document describes the database indexes created to optimize query performance for the Go server application.

## Overview

The indexes were designed based on analysis of query patterns in the repository layer. They focus on optimizing the most common and performance-critical queries.

## Index Categories

### 1. Client + Date Range Queries

These are the most common query patterns in the application, used for retrieving holdings and transactions for specific clients within date ranges.

#### Holdings Table

- `idx_holdings_client_date_range`: Optimizes `WHERE client_id = $1 AND date BETWEEN $2 AND $3`
- `idx_holdings_date_client`: Optimizes date-first queries and range scans

#### Transactions Table

- `idx_transactions_client_date_range`: Optimizes `WHERE client_id = $1 AND date BETWEEN $2 AND $3`
- `idx_transactions_date_client`: Optimizes date-first queries and range scans

### 2. JOIN Operation Optimization

These indexes optimize queries that join holdings/transactions with assets and categories.

#### Holdings Table

- `idx_holdings_asset_date`: Optimizes JOINs with assets table
- `idx_holdings_client_asset_date`: Optimizes complex queries with client, asset, and date filters

#### Transactions Table

- `idx_transactions_asset_date`: Optimizes JOINs with assets table
- `idx_transactions_client_asset_date`: Optimizes complex queries with client, asset, and date filters

### 3. Aggregation Query Optimization

These indexes optimize GROUP BY operations for reporting and analytics.

- `idx_holdings_date_for_aggregation`: Optimizes `GROUP BY DATE(date)` operations
- `idx_transactions_date_for_aggregation`: Optimizes `GROUP BY DATE(date)` operations

### 4. Soft Delete Filtering

Partial indexes that only include non-deleted records, improving performance for active data queries.

- `idx_holdings_active_client_date`: Only includes records where `deleted = FALSE`
- `idx_transactions_active_client_date`: Only includes records where `deleted = FALSE`

### 5. Additional Optimizations

- `idx_sync_logs_date`: Optimizes date-based queries on sync logs
- `idx_assets_active_external_id`: Optimizes external ID lookups for non-deleted assets
- `idx_asset_categories_active_name`: Optimizes name lookups for non-deleted categories

## Query Patterns Optimized

### Holdings Repository

```sql
-- Optimized by idx_holdings_client_date_range
SELECT * FROM holdings WHERE client_id = $1 AND date BETWEEN $2 AND $3

-- Optimized by idx_holdings_client_asset_date
SELECT * FROM holdings h
JOIN assets a ON h.asset_id = a.id
WHERE h.client_id = $1 AND h.date BETWEEN $2 AND $3

-- Optimized by idx_holdings_date_for_aggregation
SELECT DATE(h.date), SUM(h.value) FROM holdings h
WHERE h.client_id = ANY($1) AND h.date BETWEEN $2 AND $3
GROUP BY DATE(h.date)
```

### Transactions Repository

```sql
-- Optimized by idx_transactions_client_date_range
SELECT * FROM transactions WHERE client_id = $1 AND date BETWEEN $2 AND $3

-- Optimized by idx_transactions_client_asset_date
SELECT * FROM transactions t
JOIN assets a ON t.asset_id = a.id
WHERE t.client_id = $1 AND t.date BETWEEN $2 AND $3

-- Optimized by idx_transactions_date_for_aggregation
SELECT DATE(t.date), SUM(t.total_value) FROM transactions t
WHERE t.client_id = ANY($1) AND t.date BETWEEN $2 AND $3
GROUP BY DATE(t.date)
```

### Sync Logs Repository

```sql
-- Optimized by existing idx_sync_logs_client_date
SELECT sync_date FROM sync_logs
WHERE client_id = $1 AND sync_date >= $2 AND sync_date < $3

-- Optimized by idx_sync_logs_date
SELECT * FROM sync_logs WHERE sync_date >= $1 AND sync_date <= $2
```

## Performance Benefits

1. **Faster Client Queries**: Date range queries for specific clients are now significantly faster
2. **Improved JOIN Performance**: Complex queries involving multiple tables are optimized
3. **Better Aggregation**: GROUP BY operations for reporting are faster
4. **Soft Delete Efficiency**: Queries on active data are faster due to partial indexes
5. **Reduced I/O**: Fewer disk reads required for common query patterns

## Maintenance Considerations

- These indexes will slightly increase INSERT/UPDATE/DELETE overhead
- Monitor index usage with `pg_stat_user_indexes` to ensure they're being used
- Consider dropping unused indexes if they're not providing value
- Regular VACUUM and ANALYZE operations will help maintain index performance

## Migration

The indexes are created in migration `20250120000000_add_performance_indexes.sql` and can be applied using:

```bash
goose up
```

To rollback:

```bash
goose down
```
