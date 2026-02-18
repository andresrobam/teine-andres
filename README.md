# OpenRouter Go Agent

Minimal Go agent that calls OpenRouter and supports four tools:

- `db_read`: execute read-only SQL against PostgreSQL.
- `db_modify`: execute write SQL against PostgreSQL.
- `matrix_write`: send a message to a Matrix room.
- `matrix_read`: read messages from a Matrix room.

## Configuration

Set the following environment variables:

- `TOOL_LOOP_LIMIT` (optional, default `12`)
- `DATABASE_URL` (required to load startup data and use `db_read` and `db_modify`)

OpenRouter credentials are loaded from `system_credentials` rows with these names:

- `OPENROUTER_API_KEY` (required)
- `OPENROUTER_MODEL` (required)

Matrix credentials are loaded from `system_credentials` rows with these names:

- `MATRIX_BASE_URL` (required for `matrix_read` and `matrix_write`)
- `MATRIX_ACCESS_TOKEN` (required for `matrix_read` and `matrix_write`)

## Run

```
go run .
```
