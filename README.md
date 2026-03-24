# Koolbase API

The Go API powering [Koolbase](https://koolbase.com) — a Flutter-first Backend as a Service.

## Self-hosting

### Requirements
- Docker and Docker Compose

### Quick start
```bash
git clone https://github.com/kennedyowusu/koolbase-api
cd koolbase-api
cp .env.example .env
# Edit .env with your values
docker compose up
```

The API will be available at `http://localhost:8080`.
MinIO console at `http://localhost:9001` (user: `minioadmin`, password: `minioadmin`).

### Services
| Service | Port | Description |
|---------|------|-------------|
| koolbase-api | 8080 | Go REST API |
| PostgreSQL | 5432 | Primary database |
| Redis | 6379 | Caching |
| MinIO | 9000 | S3-compatible object storage |
| MinIO Console | 9001 | Storage management UI |

### Environment variables
See `.env.example` for all available options.

### Storage
By default, self-hosted instances use MinIO for object storage. To use Cloudflare R2 or AWS S3 instead, set:
```env
S3_ENDPOINT=https://<accountID>.r2.cloudflarestorage.com  # R2
# or
S3_ENDPOINT=https://s3.us-east-1.amazonaws.com            # AWS S3
```

### Migrations
Migrations run automatically on startup via the `migrate` service.

## Cloud hosted
If you don't want to self-host, use [app.koolbase.com](https://app.koolbase.com) — free to start.

## License
MIT
