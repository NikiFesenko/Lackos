-- Migration: 000001_create_roles
-- Roles are the top-level grouping for permission policies.
-- Examples: hr_admin, recruiter, manager, employee

CREATE TABLE roles (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        UNIQUE NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
