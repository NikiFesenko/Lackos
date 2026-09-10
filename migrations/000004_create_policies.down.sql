-- Migration: 000004_create_policies (down)
DROP INDEX IF EXISTS idx_policies_lookup;
DROP TABLE IF EXISTS policies;
