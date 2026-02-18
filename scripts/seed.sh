#!/bin/bash

# Seed script for local development
# Populates the database with test credentials

set -e

echo "Seeding database with test credentials..."

docker exec -i teine-postgres psql -U owner -d teine << 'EOF'
-- Insert Matrix credentials
INSERT INTO system_credentials (name, secret) VALUES
    ('MATRIX_BASE_URL', 'https://matrix.andresrobam.ee'),
    ('MATRIX_ACCESS_TOKEN', 'matrixtest')
ON CONFLICT (name) DO UPDATE SET secret = EXCLUDED.secret;

-- Insert OpenRouter credentials
INSERT INTO system_credentials (name, secret) VALUES
    ('OPENROUTER_API_KEY', 'opentest'),
    ('OPENROUTER_MODEL', 'moonshotai/kimi-k2.5'),
    ('OPENROUTER_BASE_URL', 'https://openrouter.ai/api')
ON CONFLICT (name) DO UPDATE SET secret = EXCLUDED.secret;

-- Insert a basic identity prompt
INSERT INTO identity (title, prompt, load_order) VALUES
    ('Core Identity', 'You are a helpful AI assistant integrated with Matrix.', 1)
ON CONFLICT (title) DO UPDATE SET prompt = EXCLUDED.prompt;

-- Insert a basic self prompt
INSERT INTO self (title, prompt, load_order) VALUES
    ('Capabilities', 'I can read and send Matrix messages, and interact with a PostgreSQL database.', 1)
ON CONFLICT (title) DO UPDATE SET prompt = EXCLUDED.prompt;
EOF

echo "Database seeded successfully!"
