#!/bin/bash
set -e

# Load .env.dev if it exists
if [ -f ".env.dev" ]; then
  export $(grep -v '^#' .env.dev | xargs)
fi

DB_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSL_MODE}"

echo "Running migrations UP..."
for f in migrations/*.up.sql; do
  echo "  → $f"
  psql "$DB_URL" -f "$f"
done
echo "Done."
