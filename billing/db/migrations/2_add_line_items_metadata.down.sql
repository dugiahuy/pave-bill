-- Drop metadata column and index
DROP INDEX IF EXISTS idx_line_items_metadata;
ALTER TABLE "line_items" DROP COLUMN IF EXISTS "metadata";