# Hatchway API

The control plane for Hatchway — a deployment control platform for mobile apps.

## Stack

- **Go** + Chi router
- **PostgreSQL** + pgx
- **Redis** — bootstrap response caching
- **zerolog** — structured logging
- **golang-migrate** — database migrations

## Getting Started

```bash
# 1. Clone and copy env
cp .env.example .env

# 2. Start dependencies
docker-compose up postgres redis

# 3. Run migrations
make migrate-up

# 4. Start the API
make run
```

## Bootstrap Endpoint

```
GET /v1/bootstrap?public_key=pk_live_xxx&device_id=uuid&platform=android&app_version=2.4.3
```

Returns a single atomic payload — flags, config, and version policy.

**Rollout evaluation happens in the SDK, not the server.**
The server returns `rollout_percentage` per flag. The SDK computes:

```dart
stableHash("$deviceId:$flagKey") % 100 < flag.rolloutPercentage
```

This keeps the bootstrap response identical for all devices, enabling CDN-level caching.

## Project Structure

```
cmd/api/          → entry point
internal/
  bootstrap/      → core bootstrap handler, service, types
pkg/
  database/       → PostgreSQL connection pool
  cache/          → Redis client
  middleware/     → zerolog request logger
migrations/       → SQL migration files
```
