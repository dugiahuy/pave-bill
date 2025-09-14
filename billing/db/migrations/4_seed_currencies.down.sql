-- Remove seeded currencies
DELETE FROM currencies WHERE code IN ('USD', 'GEL');