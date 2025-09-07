DROP INDEX IF EXISTS idx_bills_workflow_id;
ALTER TABLE bills DROP COLUMN IF EXISTS workflow_id;