-- Migration: 000005_create_audit_events (down)
DROP INDEX IF EXISTS idx_audit_events_user_outcome;
DROP INDEX IF EXISTS idx_audit_events_outcome;
DROP INDEX IF EXISTS idx_audit_events_server_time;
DROP INDEX IF EXISTS idx_audit_events_user_time;
DROP TABLE IF EXISTS audit_events;
