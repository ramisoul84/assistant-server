#!/bin/bash
set -e

[ -f .env.prod ] && export $(grep -v '^#' .env.prod | xargs)

echo "Running migrations UP..."

for f in migrations/*.up.sql; do
  echo "  → $f"
  docker exec -i postgres psql -U "$DB_USER" -d "$DB_NAME" < "$f"
done

echo "Done."