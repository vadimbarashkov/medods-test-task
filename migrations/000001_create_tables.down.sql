DROP TRIGGER IF EXISTS refresh_tokens_update_updated_at ON refresh_tokens;

DROP FUNCTION IF EXISTS update_timestamp();

DROP TABLE IF EXISTS refresh_tokens;
