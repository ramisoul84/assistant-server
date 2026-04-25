#!/bin/bash
set -e

echo "── Pulling latest code ──"
git pull origin main

echo "── Building and restarting containers ──"
docker compose down
docker compose up --build -d

echo "── Cleaning old images ──"
docker image prune -f

echo "── Status ──"
docker compose ps