# Force Refresh Functionality

## Overview

The `SyncAccount` endpoint now supports a force refresh feature that allows you to delete all existing holdings, transactions, and sync logs for a specific client and date range before performing a fresh sync.

## Usage

### HTTP Header

To enable force refresh, add the following header to your request:

```
x-force-refresh: true
```

### Example Request

```bash
curl -X POST "https://your-api.com/api/accounts/sync" \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -H "x-force-refresh: true" \
  -d '{
    "accountID": "your-account-id",
    "startDate": "2024-01-01",
    "endDate": "2024-01-31"
  }'
```

## What Happens During Force Refresh

When the `x-force-refresh: true` header is present, the system will:

1. **Delete Holdings**: Remove all holdings records for the specified client and date range
2. **Delete Transactions**: Remove all transaction records for the specified client and date range
3. **Delete Sync Logs**: Remove all sync log entries for the specified client and date range
4. **Perform Fresh Sync**: Execute the normal sync process to fetch and store new data

## Database Operations

The force refresh performs the following database operations:

```sql
-- Delete holdings
DELETE FROM holdings
WHERE client_id = $1 AND date BETWEEN $2 AND $3

-- Delete transactions
DELETE FROM transactions
WHERE client_id = $1 AND date BETWEEN $2 AND $3

-- Delete sync logs
DELETE FROM sync_logs
WHERE client_id = $1 AND sync_date >= $2 AND sync_date <= $3
```

## Use Cases

- **Data Correction**: When you need to correct data that was incorrectly synced
- **Fresh Start**: When you want to ensure you have the most up-to-date data
- **Debugging**: When investigating sync issues and need to start from scratch
- **Data Migration**: When migrating to a new data structure

## Security Considerations

- Force refresh is a destructive operation that permanently deletes data
- Use with caution in production environments
- Consider implementing additional authorization checks if needed
- Log all force refresh operations for audit purposes

## Error Handling

If force refresh fails:

- The operation will return an error and not proceed with the sync
- No data will be deleted if any part of the force refresh fails
- Check logs for detailed error information

## Implementation Details

The force refresh functionality is implemented across multiple layers:

1. **Handler Layer**: Checks for `x-force-refresh` header
2. **Controller Layer**: Calls force refresh service if header is present
3. **Service Layer**: Orchestrates the deletion operations
4. **Repository Layer**: Performs the actual database deletions

## Testing

To test the force refresh functionality:

1. First sync some data normally
2. Make a request with `x-force-refresh: true`
3. Verify that the old data is deleted and new data is synced
4. Check that sync logs are properly updated
