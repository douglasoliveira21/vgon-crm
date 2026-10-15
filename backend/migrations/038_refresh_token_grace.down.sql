DROP INDEX IF EXISTS idx_refresh_tokens_previous_token;

ALTER TABLE refresh_tokens
    DROP COLUMN IF EXISTS previous_token,
    DROP COLUMN IF EXISTS previous_token_expires_at;
