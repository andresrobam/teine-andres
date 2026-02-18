-- migrate:up
INSERT INTO system_credentials (name, secret)
VALUES
    ('OPENROUTER_API_KEY', ''),
    ('OPENROUTER_MODEL', '')
ON CONFLICT (name) DO NOTHING;

-- migrate:down
DELETE FROM system_credentials
WHERE name IN ('OPENROUTER_API_KEY', 'OPENROUTER_MODEL');
