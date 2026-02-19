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

if [ -z "$DATABASE_PASSWORD_AGENT" ]; then
    echo "Error: DATABASE_PASSWORD_AGENT is required"
    exit 1
fi

# --- SSH key management ---
# Default EXEC_SSH_KEY_PATH if not set
EXEC_SSH_KEY_PATH="${EXEC_SSH_KEY_PATH:-/app/data/id_ed25519}"
export EXEC_SSH_KEY_PATH

if [ -f "$EXEC_SSH_KEY_PATH" ]; then
    echo "Using existing SSH key at $EXEC_SSH_KEY_PATH"
else
    echo "No SSH key found at $EXEC_SSH_KEY_PATH, generating a new Ed25519 keypair..."
    mkdir -p "$(dirname "$EXEC_SSH_KEY_PATH")"
    ssh-keygen -t ed25519 -f "$EXEC_SSH_KEY_PATH" -N "" -q
    echo "SSH keypair generated."
fi

chmod 600 "$EXEC_SSH_KEY_PATH"

echo "--- Public key (OpenSSH format) ---"
cat "${EXEC_SSH_KEY_PATH}.pub"
echo "------------------------------------"

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
