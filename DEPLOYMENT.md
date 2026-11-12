# CodexSecure API — Deployment (Coolify)

This repository (`web3connected/codexsecure`, branch `main`) is the CodexSecure Go API.
The previous Next.js developer portal is preserved on branch `archive/nextjs-portal`
and tag `nextjs-portal-final` (commit `00d76d3`).

## Coolify settings

| Setting | Value |
|---|---|
| Build pack | Dockerfile |
| Dockerfile location | `/Dockerfile` |
| Exposed port | `8084` |
| Health check path | `/api/v1/status` (always mounted, no auth) |
| Optional extra probes | `/health`, `/ready` (DB check) — only when `ENABLE_WEB_ROUTES=true`, which also exposes unauthenticated `/metrics` |

The process listens on `SERVER_HOST:SERVER_PORT` (image defaults `0.0.0.0:8084`).

## Environment variables

All names are listed in [`.env.example`](.env.example). Set real values in Coolify only.

Required for a working service:

- `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_SSL_MODE` — codex_zones PostgreSQL.
  The process exits at startup if `DB_USER`/`DB_PASSWORD` are missing or the DB is unreachable.
- `SERVICE_API_KEY` — **must be set**; when empty, `ServiceKeyMiddleware` lets requests through.
- `CODEXSECURE_GATEWAY_ASSERTION_KEYS` — gateway JWS verifier keys; channel routes refuse every request without it.
- `CODEXHASH_API_URL`, `CODEXTIME_API_URL` — upstream services.
- `PIPELINE_API_TOKEN` — Stage 7 recording token.
- `MCP_ENABLED=false` unless codex-zone-mcp is deployed (code default is `true`).

## Database

The schema is owned by Web3Codex. This service runs no migrations. The target
codex_zones database must already contain every migration the current Go code
expects (communication channels C1–C3A, C10, and the C11-H ChannelHash v2 change).

## Local build check

```bash
docker build -t codexsecure-api:local .
docker run --rm -p 8084:8084 --env-file .env codexsecure-api:local
curl -s http://localhost:8084/api/v1/status
```

`docker-compose.yml` is the local development stack only (external `codex-net` network).
