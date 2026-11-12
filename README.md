# CodexSecure API

Stage 2 Receive: [communication retrieval contract and local verification](docs/STAGE2_COMMUNICATION_RETRIEVAL.md). Source retrieval tests pass; real enrollment and gateway authentication remain prerequisites.

## Verified implementation note — October 20, 2026

CodexSecure owns the isolated `codex_zones` service database; Laravel owns its separate application database and communicates through the SDK/API. The current physical schema contains `pagemeta_packages` and its `migrations` history. Existing dedicated migration tooling is hosted in the Web3Codex workspace under a separate schema-admin connection; Go does not execute DDL at startup.

`/health` is process liveness; `/ready` checks database connectivity; `/api/v1/policy/zones` exposes the current configuration-backed definitions for the Z0 Boundary plus the twelve zones Z1–Z12. `/api/v1/zones/health` is a distinct SQL registry endpoint: it reads `security_zones`, a relation absent from the current database, and returns HTTP 500. Its success path labels active rows healthy; it does not measure harmonization, communication approval, or security outcomes. No reviewed migration establishes that table, and no evidence establishes that it should be deleted or created automatically.

PageMeta recording/recognition is implemented and tested; learned harmonization and communication authorization remain separate development work. The older schema lists, planned routes and implementation claims below are historical/design material, not a current provisioning recipe. Do not create historical tables, repoint the database, or change the zone-health contract without the required review. See the Web3Codex report `docs/CodexSecure/CODEXSECURE_STABILITY_REVIEW.md` for source references, test results and proposals.

**Language:** Go  
**Purpose:** Network boundary layer for zone-based security and routing  
**Version:** 1.0.0  
**Created:** February 8, 2026

---

## Overview

CodexSecure acts as a **7-stage security pipeline** between the external world and your application. The MCP server sits in **Stage 1 only** as an ingress gateway and hypothesis generator. It can say "this looks like Z5," but that is only a provisional guess. The request still has to pass through normalization, context, classification, coherence, zone alignment, and execution before the zone is actually decided.

> **Scope note (2026-10-25):** the diagram and Stage 1 description below describe the Go `/api/v1/pipeline/analyze` diagnostic pipeline. They are not the authoritative Seven-Fold Process stage map; see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) §4 and §8. In the Seven-Fold Process, Stage 1 (Initialize, run by the Web3Codex SDK) currently establishes physical alignment and registry alignment ([docs/ZONE_ALIGNMENT.md](docs/ZONE_ALIGNMENT.md)). Registry alignment is not per-request authorization.

## Architecture

```
External Request
    ↓
[Stage 1 MCP Gateway - ingress preservation + zone hypothesis]
    ↓
[Stages 2-5 CodexSecure - normalize, context, classify, cohere]
    ↓
[Stage 6 Zone Align - final zone decision]
    ↓
[Stage 7 Execute/Record]
    ↓
Application Logic
```

## Stage 1 MCP Role

The MCP server is installed beside the Go app as a gateway service.

- Every inbound request starts at the `Z0` Boundary.
- `Z0` is not a zone. It is the entry boundary and unassigned state. Quarantine is a separate security decision that can keep a request there.
- It receives ingress signals.
- It preserves request context and proposes a destination zone hypothesis in metadata.
- It can early-flag obvious `Z0` quarantine candidates.
- It does **not** decide the final zone.
- Its hypothesis is later reinforced or penalized based on the final Stage 6 result.
- Over repeated cycles this reward/penalty model is intended to drive what the project calls **phase lock**.

In short:

- MCP says: "This request is entering through the `Z0` Boundary, and we think it may belong to Z5."
- CodexSecure pipeline says: "After all 7 stages, the answer is Z5" or "the answer is not Z5."
- The MCP hypothesis is rewarded for a correct guess and reduced for a wrong guess.

For security handling, there is one deliberate exception:

- if Stage 1 identifies an obvious `Z0` quarantine case, it may keep the request in `Z0` and short-circuit into quarantine handling before the normal 7-stage fulfillment path continues.

That is different from deciding an operational zone. A request held for quarantine stays at the `Z0` Boundary, but the Boundary itself is the unassigned state, not a quarantine zone.

## Core Concepts

### 1. Pattern Detection
- Analyzes incoming data patterns (email, phone, payment card, etc.)
- **Hashes the pattern, NOT the actual data**
- Stores pattern hashes in `codex_zones` database
- Example: Email pattern in login context → Hash → Zone 2 alignment

### 2. Intent Derivation
- Determines intent from action context
- Uses naming conventions and request structure
- Maps intent to appropriate zone
- Example: Login action + email pattern → Zone 2 (Identity)

### 3. Zone Architecture
- **Z0:** Boundary — entry boundary and unassigned state; outside the zone structure, not a thirteenth zone
- **Z1-Z12:** 12 operational zones, each a class of internet traffic (not a user role or site department)
- **Fractal Sub-zones:** Each zone can contain 12 sub-zones (Z2:Z01 through Z2:Z12)
- **Zone keys:** Z0 boundary (not a zone), Z1 public, Z2 identity, Z3 operations, Z4 protected, Z5 processing, Z6 logging, Z7 communications, Z8 payments, Z9 analytics, Z10 streaming, Z11 synchronization, Z12 administration (source of truth: `policy.DefaultZones()`)
- **Zone alignment:** registry (`POST /api/v1/policy/alignment`, `GET /api/v1/policy/registry`) and definitions (`POST /api/v1/policy/alignment/definitions`, `GET /api/v1/policy/definitions`) — see [docs/ZONE_ALIGNMENT.md](docs/ZONE_ALIGNMENT.md)

### 4. Middleware Independence
- Each zone has its own middleware stack
- Sub-zones inherit parent middleware + add their own
- Full configurability at every level
- Example: Z8 (Payments) middleware ≠ Z8:Z05 (Payment Verification) middleware

## Technology Stack

- **Language:** Go (chosen for performance, concurrency, networking capabilities)
- **Database:** PostgreSQL (codex_zones)
- **Framework:** Gin (HTTP routing and middleware)
- **PostgreSQL Driver:** pgx/v5
- **Pattern Hashing:** crypto/sha256
- **MCP Integration:** HTTP bridge to Node.js MCP servers

## Database Schema

### Tables
1. `harmonic_zones` - Zone definitions and configurations
2. `zone_validations` - Pattern validation records
3. `resonance_logs` - Zone routing events
4. `tiu_events` - Trust Integrity Update events
5. `zone_patterns` - Pattern hash mappings

## Key Features

### Pattern-Based Routing
- Real-time pattern detection
- Hash-based pattern matching
- Intent-aware zone selection
- Learning system for new patterns

### Zone Management
- RESTful API for zone configuration
- Real-time zone metrics
- Zone health monitoring
- Pattern hash registration

### Security
- No raw data storage (patterns only)
- Hash-based identification
- Zone-level access control
- Middleware chain validation

## API Endpoints (Planned)

### Zone Management
- `GET /api/zones` - List all zones
- `GET /api/zones/:id` - Get zone details
- `PUT /api/zones/:id` - Update zone configuration
- `GET /api/zones/:id/health` - Zone health status

### Pattern Management
- `POST /api/patterns/detect` - Detect pattern in data
- `POST /api/patterns/register` - Register new pattern hash
- `GET /api/patterns/:hash` - Get pattern info
- `POST /api/patterns/match` - Match data against patterns

### Routing
- `POST /api/route` - Determine zone for request
- `GET /api/route/intent/:action` - Get zone by intent

### Metrics
- `GET /api/metrics/zones` - Zone distribution metrics
- `GET /api/metrics/patterns` - Pattern detection stats
- `GET /api/metrics/validations` - Validation metrics

## Development Roadmap

### Phase 1: Foundation
- [x] Project structure setup
- [ ] Go module initialization
- [ ] PostgreSQL connection
- [ ] Basic HTTP server (Gin)
- [ ] Configuration management

### Phase 2: Core Engine
- [ ] Pattern detection engine
- [ ] Hash generation system
- [ ] Intent derivation logic
- [ ] Zone router implementation

### Phase 3: Zone System
- [ ] Zone middleware orchestrator
- [ ] Fractal sub-zone support
- [ ] Middleware chain execution
- [ ] Zone configuration loader

### Phase 4: Integration
- [ ] MCP server bridge
- [ ] Laravel integration helpers
- [ ] API endpoints implementation
- [ ] Metrics collection

### Phase 5: Production
- [ ] Docker containerization
- [ ] Health checks
- [ ] Monitoring & logging
- [ ] Performance optimization

## Directory Structure

```
codexsecure_api/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── database/
│   │   ├── postgres.go          # PostgreSQL connection
│   │   └── migrations/          # Database migrations
│   ├── engine/
│   │   ├── pattern.go           # Pattern detection engine
│   │   ├── intent.go            # Intent derivation
│   │   └── hash.go              # Pattern hashing
│   ├── router/
│   │   ├── zone_router.go       # Zone routing logic
│   │   └── middleware.go        # Middleware orchestration
│   ├── api/
│   │   ├── handlers/            # HTTP handlers
│   │   └── routes.go            # Route definitions
│   └── models/
│       ├── zone.go              # Zone model
│       └── pattern.go           # Pattern model
├── pkg/
│   └── mcp/
│       └── bridge.go            # MCP server integration
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml
├── docs/
│   └── ARCHITECTURE.md
├── go.mod
├── go.sum
└── README.md
```

## Configuration

```yaml
# config.yaml (example)
server:
  port: 8083
  host: "0.0.0.0"

database:
  host: "172.104.24.82"
  port: 5432
  database: "codex_zones"
  username: "${DB_USERNAME}"
  password: "${DB_PASSWORD}"
  ssl_mode: "require"

zones:
  router_id: "Z0"
  count: 12
  fractal_depth: 12

patterns:
  cache_size: 10000
  hash_algorithm: "sha256"

mcp:
  enabled: true
  node_server_url: "http://localhost:3000"
```

## Why Go?

1. **Performance** - Compiled, fast execution for network layer
2. **Concurrency** - Goroutines handle concurrent pattern matching
3. **Networking** - Built-in HTTP/2, excellent proxy capabilities
4. **Type Safety** - Strong typing for zone routing logic
5. **Deployment** - Single binary, easy Docker container
6. **Learning** - Simpler than expected, great for this use case

## Next Steps

1. Initialize Go module
2. Setup basic Gin server
3. Implement PostgreSQL connection
4. Build pattern detection engine
5. Create zone router logic
6. Integrate with existing Laravel services

---

**Status:** Initial Setup Phase  
**Last Updated:** February 8, 2026
