-- Migration: 000001_initial_schema (down)
-- Drops all tables in reverse dependency order.

DROP INDEX IF EXISTS idx_audit_events_outcome;
DROP INDEX IF EXISTS idx_audit_events_server_time;
DROP INDEX IF EXISTS idx_audit_events_user_time;

DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS policies;
DROP TABLE IF EXISTS downstream_servers;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS roles;
