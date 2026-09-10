-- Migration: 000002_create_users (down)
DROP INDEX IF EXISTS idx_users_role_id;
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;
