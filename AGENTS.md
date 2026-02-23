# Agent Guide

## Purpose
This file guides LLM agents working in this repository. It covers the overall project layout, high-level coding guidance, and database migration/permission rules.

## Project map
- main.go: OpenRouter loop, message handling, and tool execution.
- dbmodule/: database access and repositories.
- matrixmodule/: Matrix client and tools.
- execmodule/: remote exec tool integration.
- db/migrations/: schema changes.
- prompts/: system and database guidance documents.

## Coding guidance
- Follow existing Go patterns (context timeouts, error handling, gofmt).
- Prefer repository helpers for database writes.
- Keep OpenRouter request/response handling consistent with current message structs.
- Avoid logging secrets or credentials.

## Database migrations
- Always add new migration files; never edit existing migrations.
- Include both up and down sections.
- Use clear, timestamped filenames matching the existing migration pattern.
- When adding tables or columns, update prompts/DATABASE.md accordingly.

## Permissions and access
- Owner user: full access to all tables.
- Agent user: read-only access to sensitive tables.
- Conversations are stored in conversations and conversation_messages; agent is read-only, owner is full access.
- Consult prompts/DATABASE.md for table-by-table access rules.
