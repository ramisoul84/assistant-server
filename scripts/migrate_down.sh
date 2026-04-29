#!/bin/bash
set -e

if [ -f ".env.prod" ]; then
  export $(grep -v '^#' .env.prod | xargs)
fi

CONTAINER_NAME="postgres"

# Check if the container is running
if ! docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
  echo "Error: PostgreSQL container '$CONTAINER_NAME' is not running"
  exit 1
fi

echo "Running migrations DOWN (reverse order) in container: $CONTAINER_NAME"

for f in $(ls migrations/*.down.sql | sort -r); do
  echo "  → $f"
  docker exec -i "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME" -f -
done < <(cat migrations/*.down.sql | sort -r)

echo "Done."