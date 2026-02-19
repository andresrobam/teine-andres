#!/bin/sh
set -e

# Check required environment variables
if [ -z "$DATABASE_URL" ]; then
    echo "Error: DATABASE_URL is required"
    exit 1
fi

if [ -z "$DATABASE_PASSWORD_OWNER" ]; then
    echo "Error: DATABASE_PASSWORD_OWNER is required"
    exit 1
fi

# Construct the full database URL with owner credentials
# Parse the base URL and insert owner:password@ after the protocol
# Handle both postgres:// and postgresql:// schemes
DBMATE_URL=$(echo "$DATABASE_URL" | sed -E "s|^postgres://|postgres://owner:${DATABASE_PASSWORD_OWNER}@|; s|^postgresql://|postgresql://owner:${DATABASE_PASSWORD_OWNER}@|")

export DATABASE_URL="$DBMATE_URL"

# Run migrations
echo "Running database migrations..."
dbmate up

# Run the main application
exec ./teine-andres "$@"
