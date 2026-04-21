#!/bin/bash
set -e

if [ -f ".env.dev" ]; then
  export $(grep -v '^#' .env.dev | xargs)
fi

DB_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSL_MODE}"

echo "Running migrations DOWN (reverse order)..."
for f in $(ls migrations/*.down.sql | sort -r); do
  echo "  → $f"
  psql "$DB_URL" -f "$f"
done
echo "Done."
