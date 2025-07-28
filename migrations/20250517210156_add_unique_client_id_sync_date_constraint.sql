-- +goose Up
-- +goose StatementBegin

-- Add unique constraint to prevent duplicate client_id and sync_date combinations
ALTER TABLE sync_logs
ADD CONSTRAINT unique_client_sync_date UNIQUE (client_id, sync_date);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove the unique constraint
ALTER TABLE sync_logs
DROP CONSTRAINT IF EXISTS unique_client_sync_date;
-- +goose StatementEnd
