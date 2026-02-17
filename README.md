# OpenRouter Go Agent

Minimal Go agent that calls OpenRouter and supports four tools:

- `db_read`: execute read-only SQL against PostgreSQL.
- `db_modify`: execute write SQL against PostgreSQL.
- `matrix_write`: send a message to a Matrix room.
- `matrix_read`: read messages from a Matrix room.

## Configuration

Set the following environment variables:

- `OPENROUTER_API_KEY` (required)
- `OPENROUTER_MODEL` (optional, default `openai/o3-mini`)
- `OPENROUTER_BASE_URL` (optional, default `https://openrouter.ai/api/v1`)
- `TOOL_LOOP_LIMIT` (optional, default `12`)
- `DATABASE_URL` (required to use `db_read` and `db_modify`)
- `MATRIX_BASE_URL` (required to use `matrix_read` and `matrix_write`)
- `MATRIX_ACCESS_TOKEN` (required to use `matrix_read` and `matrix_write`)

## Run

```
go run .
```
