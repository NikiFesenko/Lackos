-- seed_dev.sql — Development seed data.
-- Run manually: psql $POSTGRES_DSN -f migrations/seed_dev.sql
-- NOT run automatically by the migrate service (it only runs numbered migrations).

-- ── Roles ─────────────────────────────────────────────────────────────────────
INSERT INTO roles (id, name, description) VALUES
    ('00000000-0000-0000-0000-000000000001', 'hr_admin',  'Full HR administrator access'),
    ('00000000-0000-0000-0000-000000000002', 'recruiter', 'Recruiter — candidate pipeline access only'),
    ('00000000-0000-0000-0000-000000000003', 'manager',   'People manager — own team data only'),
    ('00000000-0000-0000-0000-000000000004', 'employee',  'Individual employee — self-service only')
ON CONFLICT (name) DO NOTHING;

-- ── Sample user (hr_admin) ────────────────────────────────────────────────────
INSERT INTO users (id, email, display_name, role_id) VALUES
    (
        '00000000-0000-0000-0001-000000000001',
        'alice@example.com',
        'Alice Admin',
        '00000000-0000-0000-0000-000000000001'
    )
ON CONFLICT (email) DO NOTHING;

-- ── Mock downstream MCP server ────────────────────────────────────────────────
INSERT INTO downstream_servers (id, name, base_url, auth_type, auth_secret_ref, tool_manifest) VALUES
    (
        '00000000-0000-0000-0002-000000000001',
        'mock-bamboohr',
        'http://mockdownstream:9000',
        'api_key',
        'env:BAMBOOHR_API_KEY',
        '[
            {"name": "get_employee_record",  "description": "Fetch a single employee record by ID"},
            {"name": "list_employees",        "description": "List all employees with optional filters"},
            {"name": "list_pto_requests",     "description": "List PTO requests for an employee"},
            {"name": "update_candidate_stage","description": "Move a candidate to a new pipeline stage"},
            {"name": "get_salary_info",       "description": "Retrieve compensation data — restricted"}
        ]'::jsonb
    )
ON CONFLICT (name) DO NOTHING;

-- ── Sample policies ───────────────────────────────────────────────────────────
-- hr_admin: can call everything, no redaction.
INSERT INTO policies (role_id, downstream_server_id, tool_name, is_allowed, redact_fields, max_calls_per_minute) VALUES
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0002-000000000001', 'get_employee_record',   TRUE, '{}',             60),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0002-000000000001', 'list_employees',         TRUE, '{}',             60),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0002-000000000001', 'list_pto_requests',      TRUE, '{}',             60),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0002-000000000001', 'update_candidate_stage', TRUE, '{}',             60),
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0002-000000000001', 'get_salary_info',        TRUE, '{}',             30),

-- recruiter: can read employees and manage candidates, but salary is redacted.
    ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0002-000000000001', 'get_employee_record',   TRUE, '{"salary","ssn"}', 30),
    ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0002-000000000001', 'list_employees',         TRUE, '{"salary","ssn"}', 30),
    ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0002-000000000001', 'update_candidate_stage', TRUE, '{}',               30),
    ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0002-000000000001', 'get_salary_info',        FALSE,'{}',               0),

-- employee: self-service only — PTO requests, no salary.
    ('00000000-0000-0000-0000-000000000004', '00000000-0000-0000-0002-000000000001', 'list_pto_requests',      TRUE, '{}',              10),
    ('00000000-0000-0000-0000-000000000004', '00000000-0000-0000-0002-000000000001', 'get_employee_record',   FALSE,'{}',               0),
    ('00000000-0000-0000-0000-000000000004', '00000000-0000-0000-0002-000000000001', 'get_salary_info',        FALSE,'{}',               0)
ON CONFLICT (role_id, downstream_server_id, tool_name) DO NOTHING;
