-- Seed USD and GEL currencies
INSERT INTO currencies (code, symbol, rate, enabled) VALUES
  ('USD', '$', 1.00000000, true),
  ('GEL', '₾', 2.65000000, true)
ON CONFLICT (code) DO UPDATE SET
  symbol = EXCLUDED.symbol,
  rate = EXCLUDED.rate,
  enabled = EXCLUDED.enabled;