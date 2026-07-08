-- Rollback 033: drop the owner-key anchor and restore the users-row-only model.
-- Owner keys (user_id IS NULL) must be removed first or the NOT NULL restore fails.

DELETE FROM user_api_keys WHERE user_id IS NULL;

DROP INDEX IF EXISTS idx_user_api_keys_logto_id;

ALTER TABLE user_api_keys DROP CONSTRAINT IF EXISTS user_api_keys_anchor_check;

ALTER TABLE user_api_keys DROP COLUMN IF EXISTS logto_id;

ALTER TABLE user_api_keys ALTER COLUMN user_id SET NOT NULL;
