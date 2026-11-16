# CodexSecure API - Setup Log

## Verified implementation note — October 20, 2026

CodexSecure owns the isolated `codex_zones` service database; Laravel owns its separate application database and communicates through the SDK/API. The current physical schema contains `pagemeta_packages` and its `migrations` history. Existing dedicated migration tooling is hosted in the Web3Codex workspace under a separate schema-admin connection; Go does not execute DDL at startup.

`/health` is process liveness; `/ready` checks database connectivity; `/api/v1/policy/zones` exposes the current configuration-backed Z0 plus Z1–Z12 definitions. The legacy SQL zone endpoints (`/api/v1/zones`, `/zones/health`, `/zones/detect`, `/zones/:code`) read the retired `security_zones` table, always returned HTTP 500, and were removed on 2026-11-16 together with the startup `security_zones` probes; zone definitions come from the policy registry. The table is not recreated. Container health is `GET /api/v1/status`.

PageMeta recording/recognition is implemented and tested; learned harmonization and communication authorization remain separate development work. The older schema lists, planned routes and implementation claims below are historical/design material, not a current provisioning recipe. Do not create historical tables, repoint the database, or change the zone-health contract without the required review. See the Web3Codex report `docs/CodexSecure/CODEXSECURE_STABILITY_REVIEW.md` for source references, test results and proposals.

## Progress — Stage 1 (Initialize) and zone alignment, October 25, 2026

Architecture: one Boundary (Z0) and twelve operational zones (Z1–Z12). See `docs/ARCHITECTURE.md` §4 and §6 and `docs/ZONE_ALIGNMENT.md`.

### Completed — Web3Codex Stage 1 (Web3Codex SDK, not this repository)

Source: Web3Codex `docs/codexsecure/api/STAGE1_ZONE_ALIGNMENT.md`. On 2026-10-25 the code was confirmed present by read-only inspection. The Web3Codex tests were **not** re-run from this repository.

- Physical zone directories `routes/zones/z1`–`z12` with identity dotfiles (`.public` … `.administration`) are implemented. No Z0 directory exists, because Z0 is not a zone.
- Physical alignment is checked during route registration (`ZonedRouteLoader`, `PhysicalZoneMarker`).
- The route cache preserves the resolved physical identity (route attributes survive `route:cache`).
- Initialize is integrated with the existing Go alignment API (`ZoneAlignment` → `POST /secure/api/v1/policy/alignment`).
- The shared Seven-Fold context (`codexsecure.initialize`) stores the alignment results.
- The Zone Debug Bar (`CodexSecureDebugger`) displays Stage 1 execution and outcomes.
- Stage progression responds to the Initialize result: Receive runs only when Stage 1 passes.
- Focused tests were added (`tests/Feature/CodexSecureInitializeTest.php`) and were verified by the Web3Codex Stage 1 work.

### Completed — CodexSecure Go alignment (policy 3.0.0, this repository)

- Canonical registry consolidated in `policy.DefaultZones()`; analyzer keys, alignment and pipeline lookups resolve against it.
- Twelve operational zone identities aligned with Web3Codex (operations, processing, logging, communications, payments, streaming, synchronization, administration, …).
- Obsolete zone classification rules removed: Z0 spam keywords, Z8 "Resource Sync" composite rule, Z10 behavior and Z11 agent-mesh vocabularies.
- Compatible legacy names are supported as same-zone aliases (for example `admin` → Z12). Former names of zones whose responsibility changed (Z8, Z10, Z11) are not aliases.
- The alignment endpoint was verified against the actual Web3Codex payload, including Web3Codex's own response validator.
- Go tests passed.
- The new build was verified in a separate container. It was **not** deployed.

### Current update — Boundary naming (policy 3.1.0, this repository)

- Z0 renamed to Boundary (key `boundary`). Identifier and rules (read-only, GET only) are unchanged. `lobby` and `negative` remain Z0 aliases for existing clients.
- Documentation distinguishes the Boundary from the twelve zones, and separates physical alignment, registry alignment and request processing. The `pipeline/analyze` Stage 1 default label changed from `quarantine_envelope` to `boundary`. Actual quarantine candidates keep `quarantine_envelope`. The analyzer `zones_used` stat counts Z1–Z12 only.
- Compatibility and alignment behavior were reverified in a separate container. With Web3Codex's payload at that time (Z0 `lobby`), Go reports aligned through the alias, but **Web3Codex's strict validator rejects the response**. Web3Codex must submit `boundary` before or together with deployment (done locally on 2026-10-25; see below). See `docs/ZONE_ALIGNMENT.md` → Web3Codex compatibility.

### Stage 1 — Initialize: COMPLETE (local development, 2026-10-25)

- Web3Codex now submits Z0 `boundary`, and its Z0 definition matches `DefaultZones()`. This change was made in Web3Codex, not in this repository.
- The local development `codexsecure-api` was rebuilt from this source and restarted alone. It serves policy **3.1.0**. Environment, network, port and database are unchanged. Rollback image: `codexsecure-api:pre-boundary-20261025` (2.0.0).
- Signatures: Web3Codex `local_signature`, Go `expected_signature` and Go `signature` are all `5eca793c1a786b864899595b2ae784a433e795efaa9a70c24cdcfba8ecbaaff5`.
- The real gateway response is `aligned: true` with 13 identifiers matched (Boundary plus Z1–Z12) and no `equivalents` or differences. All twelve operational zones are aligned, and the Boundary identity is aligned separately. It passes Web3Codex `validResult()`.
- Web3Codex Initialize was run: Z1 (`/pages/home`), Z2 (`/login`) and Z12 (`/admin/blog/categories`) routes have valid physical alignment, are aligned and continue to Receive. Z12 ran through in-process request handling because no authenticated browser session was available. Repeat requests reuse the cached result, and the outcome is the same with `route:cache` in place.
- `go test ./...` passes; `go vet ./...` is clean; the integration-tagged tests compile.
- Web3Codex (reported there, not re-run from here): 717 tests, 618 passed, 82 skipped, 16 failed, 1 error. The failures and the error predate this change. The PostgreSQL suite was not run. `npm run build` succeeds.
- Status: local development only. No production deployment. This directory is not a git repository, and the Web3Codex Stage 1 code was not committed as part of the Stage 1 completion report. Details: Web3Codex `docs/codexsecure/api/STAGE1_ZONE_ALIGNMENT.md`.

### Remaining work

- Production deployment of the 3.1.0 build (requires separate authorization), followed by verification through the gateway (`scripts/verify_zone_alignment.sh` with `ALIGNMENT_BASE_URL=…/secure`).
- Stage 2 — Receive: the next development phase; not started.
- Per-communication alignment and request-level communication recognition.
- MCP and harmonization integration.
- Web3Codex side (tracked there): older Web3Codex test failures, and later stages can report `passed` on a redirected request (for example the Z12 request redirected to `/admin/login`).
- Out of alignment scope, still open (Go):
  - `/api/…` paths lock to Z3 in the analyzer, so `/api/admin` gets Z3 rules.
  - Analyzer-unassigned paths fall back to `public` in `pipeline/analyze`.
  - MCP and page schema are not enabled in this runtime, so `pipeline/analyze` Stage 1 always proposes Z0 and the pipeline matrix fails on production and new builds alike.
  - The `/zones` endpoints query the absent `security_zones` table. The rebuilt API also logs `request_logs` as missing, so endpoint-hit logging is skipped.
  - The startup log reports app `version 2.0.0` while the policy is 3.1.0.
  - There is no per-request zone-decision endpoint (Web3Codex contract gap).

---

**Date:** February 8, 2026  
**Status:** Initial Foundation Complete ✅

---

## What We've Built

### 1. Project Structure ✅

```
codexsecure_api/
├── cmd/server/main.go          # Application entry point with graceful shutdown
├── internal/
│   ├── config/config.go        # Configuration management (env vars)
│   └── models/
│       ├── zone.go             # Zone data structures
│       └── pattern.go          # Pattern detection structures
├── docs/ARCHITECTURE.md        # Complete architecture documentation
├── go.mod                      # Go module definition
├── .env.example                # Environment variable template
├── .gitignore                  # Git ignore rules
└── README.md                   # Project overview
```

### 2. Core Documentation ✅

**README.md:**
- Overview of CodexSecure purpose
- Architecture diagram
- Technology stack justification (Why Go?)
- Development roadmap
- Configuration guide

**ARCHITECTURE.md:**
- Detailed system architecture
- Data flow diagrams
- Component descriptions
- Database schema
- API design
- Security considerations
- Performance targets

### 3. Configuration System ✅

**config.go:**
- Loads from environment variables
- Supports .env files
- Validates required fields
- Provides typed configuration structs:
  - Server (port, host, env)
  - Database (PostgreSQL connection)
  - Zone (router, count, fractal depth)
  - Pattern (cache size, hash algorithm)
  - MCP (integration settings)
  - Logging (level, format)

### 4. Data Models ✅

**zone.go:**
- `Zone` - Main zone structure with middleware stack
- `ZoneConfig` - Zone configuration
- `ZoneHealth` - Health monitoring
- `ZoneMetrics` - Performance metrics

**pattern.go:**
- `Pattern` - Pattern detection and mapping
- `PatternType` - Enum for pattern types
- `IntentType` - Enum for intent types
- `PatternDetectionRequest/Result` - API structures

### 5. Application Entry Point ✅

**main.go:**
- Graceful startup and shutdown
- Signal handling (Ctrl+C, SIGTERM)
- Logging setup with zerolog
- Placeholder for initialization steps

---

## Technology Decisions

### Why Go?

1. **Performance** - Compiled language, fast execution for network layer
2. **Concurrency** - Goroutines perfect for concurrent pattern matching
3. **Networking** - Built-in HTTP/2, excellent for proxying
4. **Type Safety** - Strong typing prevents routing errors
5. **Deployment** - Single binary, easy Docker containerization
6. **Simplicity** - Easier to learn than expected
7. **Ecosystem** - Great libs for PostgreSQL (pgx), HTTP (gin), logging (zerolog)

### Key Libraries

- **Gin** - HTTP web framework with middleware support
- **pgx/v5** - High-performance PostgreSQL driver
- **zerolog** - Fast structured logging
- **godotenv** - Environment variable loading

---

## Architecture Decisions

### 1. Pattern Hashing, Not Data Storage

**Decision:** Store SHA-256 hashes of pattern types, not actual data  
**Rationale:**
- Privacy-first design
- GDPR/compliance friendly
- Reduces storage requirements
- Focuses on pattern recognition, not data collection

**Example:**
```
Input: "user@example.com" in login context
Store: sha256("EMAIL_PATTERN:login") → Zone Z2
NOT storing: "user@example.com"
```

### 2. Fractal Zone System (12 sub-zones per parent)

**Decision:** Each zone (Z1-Z12) can contain 12 sub-zones (Z1:Z01...Z1:Z12)  
**Rationale:**
- Granular security control
- Independent middleware per sub-zone
- Scalable architecture
- Allows specific routing (e.g., Z2:Z04 for 2FA specifically)

### 3. Z0 as Router, Not Zone

**Decision:** Z0 is intake/classification layer, not a security zone  
**Rationale:**
- Clarifies system architecture
- Z0 routes quarantine/honeypot traffic
- Actual zones are Z1-Z12 (12 zones)
- Prevents confusion in documentation

### 4. Intent-Based Routing

**Decision:** Route based on pattern + intent, not just URL  
**Rationale:**
- More intelligent than path-based routing
- Handles multiple actions on same endpoint
- Supports API versioning naturally
- Enables learning system for new patterns

---

## Next Steps

### Phase 1: Database Integration 🎯 NEXT

**Tasks:**
1. Create PostgreSQL connection pool
2. Implement database migrations
3. Build zone repository (CRUD operations)
4. Build pattern repository
5. Add connection health checks

**Files to create:**
- `internal/database/postgres.go`
- `internal/database/migrations/`
- `internal/repository/zone_repo.go`
- `internal/repository/pattern_repo.go`

### Phase 2: Pattern Engine

**Tasks:**
1. Implement pattern detection algorithms
2. Build SHA-256 hashing system
3. Create pattern cache (10k patterns)
4. Add pattern type detection (email, phone, card, etc.)

**Files to create:**
- `internal/engine/pattern.go`
- `internal/engine/hash.go`
- `internal/engine/cache.go`

### Phase 3: Intent Derivation

**Tasks:**
1. Build intent analyzer
2. Implement naming convention parser
3. Create intent → zone mapping logic
4. Add context-aware intent detection

**Files to create:**
- `internal/engine/intent.go`
- `internal/engine/context.go`

### Phase 4: Zone Router

**Tasks:**
1. Implement zone routing logic
2. Build middleware orchestrator
3. Add fractal sub-zone support
4. Create zone health checker

**Files to create:**
- `internal/router/zone_router.go`
- `internal/router/middleware.go`
- `internal/router/fractal.go`

### Phase 5: API Layer

**Tasks:**
1. Setup Gin HTTP server
2. Implement API handlers
3. Add request validation
4. Create response serializers

**Files to create:**
- `internal/api/server.go`
- `internal/api/handlers/zone_handler.go`
- `internal/api/handlers/pattern_handler.go`
- `internal/api/routes.go`

### Phase 6: MCP Integration

**Tasks:**
1. Build HTTP bridge to Node MCP server
2. Implement pattern analysis integration
3. Add threat detection hooks
4. Create MCP health monitoring

**Files to create:**
- `pkg/mcp/bridge.go`
- `pkg/mcp/client.go`

### Phase 7: Testing & Production

**Tasks:**
1. Unit tests for all components
2. Integration tests
3. Docker containerization
4. Performance benchmarking
5. Documentation completion

---

## Environment Setup

### Prerequisites

- Go 1.22+
- PostgreSQL 14+ (codex_zones database)
- Docker (for containerization)

### Installation Steps

```bash
# 1. Navigate to project
cd /home/web3codex/projects/Docker/codexsecure_api

# 2. Initialize Go modules (downloads dependencies)
go mod download

# 3. Copy environment template
cp .env.example .env

# 4. Edit .env with actual credentials
nano .env
# Set DB_USER and DB_PASSWORD

# 5. Build the application
go build -o codexsecure-api cmd/server/main.go

# 6. Run
./codexsecure-api
```

### Development Mode

```bash
# Install air for hot reload (optional)
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

---

## Configuration Variables

See `.env.example` for complete list. Key variables:

```bash
# Server
SERVER_PORT=8083              # API port
SERVER_HOST=0.0.0.0           # Listen address

# Database (codex_zones)
DB_HOST=172.104.24.82         # PostgreSQL host
DB_PORT=5432                  # PostgreSQL port
DB_NAME=codex_zones           # Database name
DB_USER=your_user             # Database user
DB_PASSWORD=your_pass         # Database password

# Zone System
ZONE_COUNT=12                 # Number of parent zones
ZONE_FRACTAL_DEPTH=12         # Sub-zones per parent

# Pattern Engine
PATTERN_CACHE_SIZE=10000      # Pattern cache size
PATTERN_HASH_ALGORITHM=sha256 # Hashing algorithm
```

---

## Database Schema

### Existing Tables (Already in codex_zones)

1. **harmonic_zones** - Zone definitions
2. **zone_validations** - Validation results
3. **resonance_logs** - Zone routing events
4. **tiu_events** - Trust integrity updates

### New Tables (To be created)

1. **zone_patterns** - Pattern hash → Zone mappings
   ```sql
   CREATE TABLE zone_patterns (
       id BIGSERIAL PRIMARY KEY,
       pattern_hash VARCHAR(64) NOT NULL UNIQUE,
       pattern_type VARCHAR(50) NOT NULL,
       intent VARCHAR(100) NOT NULL,
       zone_code VARCHAR(10) NOT NULL,
       confidence FLOAT DEFAULT 1.0,
       is_approved BOOLEAN DEFAULT false,
       created_at TIMESTAMP DEFAULT NOW(),
       updated_at TIMESTAMP DEFAULT NOW()
   );
   ```

---

## Testing the Setup

### 1. Check Go Installation
```bash
go version
# Should show: go version go1.22+ ...
```

### 2. Verify Dependencies
```bash
cd /home/web3codex/projects/Docker/codexsecure_api
go mod verify
```

### 3. Build Test
```bash
go build -o codexsecure-api cmd/server/main.go
```

### 4. Run Test (placeholder server)
```bash
./codexsecure-api
# Should show startup logs
# Press Ctrl+C to stop
```

---

## Integration Points

### With Laravel (codex_admin)

CodexSecure will integrate with Laravel services to:
1. Provide zone routing decisions via API
2. Validate patterns before processing
3. Supply zone metrics for dashboard
4. Handle zone-aware request preprocessing

**Example Laravel Usage:**
```php
// Query CodexSecure for zone routing
$response = Http::post('http://codexsecure-api:8083/api/v1/route/analyze', [
    'data' => $request->email,
    'context' => [
        'route' => '/login',
        'action' => 'auth.login'
    ]
]);

$zone = $response->json('recommended_zone'); // "Z2"
```

### With MCP Servers

CodexSecure can send pattern data to MCP servers for advanced analysis:
- Natural language intent detection
- Behavioral threat analysis
- ML-based pattern recognition

---

## Monitoring & Health Checks

### Planned Endpoints

- `GET /health` - Service health
- `GET /health/db` - Database connectivity
- `GET /health/zones` - Zone system status
- `GET /metrics` - Prometheus metrics (future)

---

## Success Criteria

✅ **Phase 1 Complete:**
- Project structure created
- Configuration system working
- Data models defined
- Documentation written
- Go modules initialized

🎯 **Next Milestone: Database Integration**
- PostgreSQL connection working
- Basic CRUD operations
- Migrations system in place

---

## Notes & Decisions Log

### Feb 8, 2026 - Initial Setup

**Decision:** Use Go for CodexSecure  
**Reason:** Performance, concurrency, networking capabilities outweigh learning curve

**Decision:** Pattern hashing instead of data storage  
**Reason:** Privacy-first, compliance-friendly, pattern-focused architecture

**Decision:** Fractal zone system (12 sub-zones per parent)  
**Reason:** Granular control, independent middleware, scalable design

**Decision:** Z0 as router, not zone  
**Reason:** Clarifies architecture, Z0 is preprocessing layer

---

## Resources & Documentation

- [Go Documentation](https://go.dev/doc/)
- [Gin Framework](https://gin-gonic.com/docs/)
- [pgx PostgreSQL Driver](https://github.com/jackc/pgx)
- [Architectural Patterns for Security](./docs/ARCHITECTURE.md)

---

**Status:** Foundation complete, ready for database integration phase.
