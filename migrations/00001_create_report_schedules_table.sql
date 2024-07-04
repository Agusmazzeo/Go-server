-- +goose Up
CREATE TABLE report_schedules (
    id SERIAL PRIMARY KEY,
    sender_id INT NOT NULL,
    recipient_organization_id INT NOT NULL,
    report_template_id INT NOT NULL,
    cron_time VARCHAR(255) NOT NULL,
    last_sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    active BOOLEAN NOT NULL
);

-- +goose Down
DROP TABLE report_schedules;
