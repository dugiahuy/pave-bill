CREATE TABLE IF NOT EXISTS "bills" (
  "id" serial PRIMARY KEY,
  "currency" varchar(4) NOT NULL,
  "status" varchar(20) NOT NULL,
  "close_reason" text,
  "error_message" text,
  "total_amount_cents" bigint,
  "start_time" timestamptz NOT NULL,
  "end_time" timestamptz NOT NULL,
  "billed_at" timestamptz,
  "idempotency_key" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT NOW(),
  "updated_at" timestamptz NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS "line_items" (
  "id" serial PRIMARY KEY,
  "bill_id" int,
  "amount_cents" bigint NOT NULL,
  "currency" varchar(4) NOT NULL,
  "description" text,
  "incurred_at" timestamptz NOT NULL,
  "reference_id" text,
  "idempotency_key" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT NOW(),
  "updated_at" timestamptz NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS "currencies" (
  "id" serial PRIMARY KEY,
  "code" varchar(4) UNIQUE,
  "symbol" varchar(4),
  "rate" decimal(18,8) NOT NULL,
  "enabled" boolean NOT NULL DEFAULT false
);

ALTER TABLE "line_items" ADD FOREIGN KEY ("bill_id") REFERENCES "bills" ("id");

-- Indexes for better query performance
CREATE INDEX idx_bills_status ON bills(status);
CREATE INDEX idx_line_items_bill_id ON line_items(bill_id);

-- Unique constraint for idempotency
CREATE UNIQUE INDEX idx_bills_idempotency_key ON bills(idempotency_key);
CREATE UNIQUE INDEX idx_line_items_idempotency_key ON line_items(idempotency_key);
