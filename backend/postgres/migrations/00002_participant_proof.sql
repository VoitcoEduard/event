ALTER TABLE participants ADD COLUMN IF NOT EXISTS secret_hash bytea;

SELECT 1;
