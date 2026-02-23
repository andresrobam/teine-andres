# teine-andres

Personal AI assistance written in Go, using the OpenRouter API to converse to it's owner over Matrix. Keeps its data in a PostgreSQL database and has access to its own VM over SSH.

## Tools

- `db_read`: execute read-only SQL against PostgreSQL.
- `db_write`: execute write SQL against PostgreSQL.
- `matrix_write`: send a message to a Matrix room.
- `matrix_read`: read messages from a Matrix room.
- `exec`: run shell commands in a remote sandbox via SSH.

## Configuration

Set the following environment variables (Docker is the default path):

Required:
- `DATABASE_PASSWORD_OWNER`
- `DATABASE_PASSWORD_AGENT`
- `OWNER_FIRST_NAME`
- `MATRIX_OWNER_ID`
- `EXEC_SSH_HOST`

Optional:
- `EXEC_SSH_USER` (default `teine-andres`)
- `EXEC_SSH_PORT` (default `22`)
- `TOOL_LOOP_LIMIT` (default `20`)
- `HTTP_TIMEOUT_SECONDS` (default `300`)

The agent reads OpenRouter and Matrix credentials from `system_credentials` in the database. Ensure these rows exist:
- `OPENROUTER_API_KEY`
- `OPENROUTER_MODEL`
- `MATRIX_BASE_URL`
- `MATRIX_ACCESS_TOKEN`

## Docker

Start Postgres and the agent:

```bash
docker-compose up -d
```

Run migrations (requires dbmate locally):

```bash
dbmate up
```

## Notes

- `EXEC_SSH_KEY_PATH` is expected at `/app/data/id_ed25519` inside the container and is already set in `docker-compose.yml`.
- The agent stores conversations in `conversations` and `conversation_messages`.
