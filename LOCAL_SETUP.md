Quick Start:
# 1. Start Postgres
docker-compose up -d
# 2. Run migrations (requires dbmate)
dbmate up
# 3. Seed test credentials
bash scripts/seed_test_credentials.sh
# 4. Run the app
go run .
Environment Variables:
export DATABASE_URL="postgres://agent:agentpass@localhost:5432/teine?sslmode=disable"
export TOOL_LOOP_LIMIT=12  # optional
