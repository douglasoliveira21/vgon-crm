DROP TABLE IF EXISTS contact_merge_history;
DROP INDEX IF EXISTS idx_audit_logs_entry_hash;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS entry_hash, DROP COLUMN IF EXISTS previous_hash;
DROP TABLE IF EXISTS data_retention_policies;
DROP TABLE IF EXISTS data_subject_requests;
DROP TABLE IF EXISTS contact_channel_consents;
DROP TABLE IF EXISTS campaign_delivery_events;
DROP TABLE IF EXISTS campaign_suppressions;
ALTER TABLE campaigns
    DROP COLUMN IF EXISTS frequency_cap_days,
    DROP COLUMN IF EXISTS approved_at,
    DROP COLUMN IF EXISTS approved_by,
    DROP COLUMN IF EXISTS approval_status,
    DROP COLUMN IF EXISTS timezone,
    DROP COLUMN IF EXISTS scheduled_at;
DROP TABLE IF EXISTS bot_execution_events;
DROP TABLE IF EXISTS bot_flow_versions;
ALTER TABLE bot_flows
    DROP COLUMN IF EXISTS published_by,
    DROP COLUMN IF EXISTS published_at,
    DROP COLUMN IF EXISTS current_version,
    DROP COLUMN IF EXISTS published_version,
    DROP COLUMN IF EXISTS lifecycle_status;
DROP TABLE IF EXISTS dead_letter_jobs;
DROP TABLE IF EXISTS durable_jobs;
