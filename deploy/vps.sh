#!/usr/bin/env bash
# One-command VPS deploy (Ubuntu/Debian): bash deploy/vps.sh
set -e
if ! command -v docker >/dev/null 2>&1; then
  curl -fsSL https://get.docker.com | sh
fi
[ -f .env ] || cp sample.env .env
echo "Edit .env now if you haven't (API_ID, API_HASH, TOKEN, STRING1, MONGO_URI, OWNER_ID, LOGGER_ID)."
# BASE_URL lets Telegram fetch the banner from this server
if ! grep -q '^BASE_URL=.' .env; then
  IP=$(curl -s https://api.ipify.org || true)
  [ -n "$IP" ] && echo "BASE_URL=http://$IP:8080" >> .env
fi
docker compose up -d --build
docker compose logs -f --tail=50
