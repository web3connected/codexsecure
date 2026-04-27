# CodexSecure SDK Standardization Setup Guide

**Last Updated:** April 21, 2026  
**Purpose:** Standardize CodexSecure integration across multi-language services (Python, Laravel, Go)

---

## Quick Reference: Service Zone Assignments

| Service | Language | Port | Zone | Assignment |
|---------|----------|------|------|-----------|
| CodexHash | Python FastAPI | 8001 | Z5 | System Execution |
| CodexTime | Python FastAPI | 8007 | Z3 | Temporal Logic |
| CodexAuth | Laravel | 8003 | Z1 | User Management |
| CodexSecure | Go | 8005 | N/A | Authority (all zones) |

---

## Architecture Pattern

### Unified HTTP SDK Pattern

All services communicate with CodexSecure via **HTTP SDK clients**, even when using different frameworks:

```
Service (Any Language)
    ↓ (HTTP SDK Client)
CodexSecure API (Go)
    ↓ (Zone validation, policy enforcement)
CodexZones Database
```

### SDK Client Responsibilities

1. **Zone Code Generation** — Convert zone ID to "Z{id}" format
2. **Zone Headers** — Attach zone context to requests
3. **Method Validation** — Optional enforcement of HTTP method policies per zone
4. **Fail-Open Default** — When CodexSecure unreachable, allow requests (safety)
5. **Timeout Handling** — Configurable request timeout with fallback

---

## Setup Pattern: CodexHash (Zone 5)

### Environment Variables

```env
CODEXSECURE_SDK_ENABLED=true
CODEXSECURE_SDK_BASE_URL=http://codexsecure-api:8005
CODEXSECURE_SDK_API_TOKEN=
CODEXSECURE_SDK_DEFAULT_ZONE=5
CODEXSECURE_SDK_TIMEOUT=5.0
CODEXSECURE_SDK_FAIL_OPEN=true
CODEXSECURE_SDK_ENFORCE_METHODS=false
```

### Middleware Registration (FastAPI)

```python
from .sdk.codexsecure import CodexSecureSDKClient

@app.middleware("http")
async def codexsecure_zone_context(request, call_next):
    """Attach Zone 5 context to all requests."""
    request.state.codex_zone = "Z5"
    
    # Optional: validate method policy
    if codexsecure_sdk.enabled:
        check = await codexsecure_sdk.validate_method(request.method)
        if not check.get("allowed", False):
            return JSONResponse(status_code=403, content={...})
    
    return await call_next(request)
```

### Docker Compose

```yaml
environment:
  - CODEXSECURE_SDK_ENABLED=true
  - CODEXSECURE_SDK_BASE_URL=http://codexsecure-api:8005
  - CODEXSECURE_SDK_DEFAULT_ZONE=5
depends_on:
  - codexsecure-api
```

---

## Setup Pattern: CodexTime (Zone 3)

**Difference from CodexHash:** Only zone assignment changes.

### Environment Variables

```env
CODEXSECURE_SDK_DEFAULT_ZONE=3  # ← Only change
# ... rest identical to CodexHash
```

### Route Method Policy (Zone 3)

Zone 3 is read-heavy; restrict mutation operations:

```
Allowed:   GET, HEAD, OPTIONS
Denied:    POST, PUT, DELETE (unless admin override)
```

---

## Setup Pattern: Laravel Services

Laravel-based services (CodexAuth, future projects) use the **PHP SDK package** (`SDK/codexsecure-sdk`):

### Installation

```json
{
  "repositories": [
    {"type": "path", "url": "../../SDK/codexsecure-sdk"}
  ],
  "require": {
    "web3codex/codexsecure-sdk": "*"
  }
}
```

### Route Registration

```php
use Web3Codex\CodexSecureSdk\Facades\CodexSecure;

// In route bootstrap:
CodexSecure::loadRoutes();
```

### Configuration

```php
// config/codexsecure-sdk.php
return [
    'enabled' => env('CODEXSECURE_SDK_ENABLED', true),
    'routes' => [
        [
            'zone' => 'Z1',
            'prefix' => 'api/auth',
            'files' => ['vendor/web3codex/codexsecure-sdk/routes/zones/Z1/user.php'],
        ],
    ],
];
```

---

## Validation Checklist (Common to All)

Every service integration should pass:

- [ ] SDK client initialized on startup (check logs)
- [ ] Zone context attached to all requests
- [ ] Zone headers in HTTP requests to CodexSecure
- [ ] Method validation works (if enabled)
- [ ] Fail-open behavior when CodexSecure unreachable
- [ ] No 500 errors from SDK initialization
- [ ] Docker Compose includes CodexSecure API in `depends_on`

---

## Files Reference

### PHP SDK Package
- Location: `SDK/codexsecure-sdk/`
- Entrypoint: `CodexSecure::loadRoutes()`
- Config: `config/codexsecure-sdk.php`
- Route Loader: `src/Support/CodexSecureRouteLoader.php`
- Zone Routes: `routes/zones/{Z#}/{name}.php`

### Python SDK Client
- Location: `Docker/{service}_api/src/sdk/codexsecure/`
- Client: `client.py` (HTTP SDK)
- Usage: `CodexSecureSDKClient.from_env()`

### Configuration
- CodexHash: `Docker/codexhash_api/.env.example`
- CodexTime: `Docker/codextime_api/.env.example`

---

## Common Issues

| Problem | Solution |
|---------|----------|
| SDK initialization fails | Verify `CODEXSECURE_SDK_BASE_URL`, CodexSecure API running |
| 403 Forbidden zone errors | Check zone assignment, method policy, or disable `CODEXSECURE_SDK_ENFORCE_METHODS` |
| Requests timeout | Increase `CODEXSECURE_SDK_TIMEOUT`, check network |
| Zone context not attached | Verify middleware registration order |

---

## Next Steps

### Immediate (This Week)
1. Review Phase 2 & Phase 3 setup plans (`tasks/task_2026-04-21.md`)
2. Approve zone assignments (Z5 for Hash, Z3 for Time)
3. Begin CodexHash implementation (if approved)

### Week 2
1. Implement CodexHash integration
2. Run validation tests
3. Begin CodexTime implementation

### Week 3+
1. Deploy both services to staging
2. Performance testing under load
3. Production rollout (CodexHash first, CodexTime second)

---

## References

- **CodexSecure Architecture:** See `_docs/CodexSecure/` directory
- **Page Schema Artifact Spec:** See `_docs/CodexSecure/PAGE_SCHEMA_ARTIFACT_SPEC.md`
- **Route-Driven Page Schema Architecture:** See `_docs/CodexSecure/ROUTE_DRIVEN_PAGE_SCHEMA_ARCHITECTURE.md`
- **Zone Model:** See `/memories/repo/codexsecure-zone-model.md`
- **Stage 7 Pipeline:** See `/memories/repo/codexsecure-go-pipeline.md`
- **Daily Task Log:** See `tasks/task_2026-04-21.md`
