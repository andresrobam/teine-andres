-- migrate:up
-- Seed identity table with default prompts
INSERT INTO identity (title, prompt, load_order)
VALUES 
    ('SYSTEM', '', 0),
    ('DATABASE', '', 1);

-- migrate:down
-- Remove seeded prompts
DELETE FROM identity WHERE title IN ('SYSTEM', 'DATABASE');
