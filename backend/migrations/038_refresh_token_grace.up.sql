-- Refresh-token rotation kept a single row per session and deleted the old
-- token before inserting the new one, with no transaction. Two tabs
-- refreshing at the same moment both passed the lookup, and the loser's
-- insert hit the unique session_id index and logged that tab out.
--
-- Keeping the previous token acceptable for a short grace window lets the
-- second request rotate successfully instead of being rejected for using a
-- token that was valid when it was sent.
ALTER TABLE refresh_tokens
    ADD COLUMN IF NOT EXISTS previous_token VARCHAR(500),
    ADD COLUMN IF NOT EXISTS previous_token_expires_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_previous_token
    ON refresh_tokens(previous_token)
    WHERE previous_token IS NOT NULL;
