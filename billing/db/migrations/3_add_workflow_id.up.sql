ALTER TABLE bills ADD COLUMN workflow_id text;

-- Index for workflow_id lookups
CREATE INDEX idx_bills_workflow_id ON bills(workflow_id);