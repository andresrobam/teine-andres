-- migrate:up
INSERT INTO system_credentials (name, secret)
VALUES
    ('MATRIX_BASE_URL', ''),
    ('MATRIX_ACCESS_TOKEN', '')
ON CONFLICT (name) DO NOTHING;

-- migrate:down
DELETE FROM system_credentials
WHERE name IN ('MATRIX_BASE_URL', 'MATRIX_ACCESS_TOKEN');
