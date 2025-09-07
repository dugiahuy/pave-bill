-- Add metadata column to line_items table for currency conversion data
ALTER TABLE "line_items" ADD COLUMN "metadata" jsonb;

-- Create index for better query performance on metadata
CREATE INDEX idx_line_items_metadata ON line_items USING GIN (metadata);