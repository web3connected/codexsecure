# CodexSecure SDK Implementation Checklist - Phase 4

**Date:** April 21, 2026  
**Status:** Implementation Phase Started  
**Target:** CodexHash & CodexTime full integration validation

---

## Phase 4: Implementation Track

### ✅ Completed Tasks

#### 1. Validation Test Suite Created
- **File:** `Docker/codexhash_api/tests/test_codexsecure_integration.py`
- **Coverage:**
  - SDK Initialization (6 tests)
  - Zone Context (5 tests)
  - Hash API (3 tests)
  - Method Validation (2 tests)
  - Fail-Open Behavior (2 tests)
  - Request Headers (3 tests)
  - Configuration (2 tests)
  - End-to-End Integration (2 tests)
- **Total:** 25 comprehensive tests

#### 2. Docker Compose Configuration
- **CodexHash Dev:** `Docker/codexhash_api/docker-compose.yml` (NEW)
  - Zone 5 configuration
  - CodexSecure API dependency
  - Network bridge setup
  - Auto-reload for development
  
- **CodexHash Prod:** `Docker/codexhash_api/docker-compose.prod.yml` (UPDATED)
  - Production CodexSecure URL
  - Healthcheck configured
  - Logging optimized
  
- **CodexTime Dev:** `Docker/codextime_api/docker-compose.yml` (UPDATED)
  - Zone 3 configuration
  - CodexSecure API dependency
  - Network bridge setup
  
- **CodexTime Prod:** `Docker/codextime_api/docker-compose.prod.yml` (UPDATED)
  - Zone 3 production settings
  - CodexSecure service added
  - Healthcheck configured

---

## Phase 4: Next Steps (Ready for Execution)

### Step 1: Verify CodexHash Environment

**Action:** Ensure CodexHash `.env` has correct values

```bash
# From Docker/codexhash_api directory
cat .env | grep CODEXSECURE_SDK
```

**Expected Output:**
```
CODEXSECURE_SDK_ENABLED=true
CODEXSECURE_SDK_BASE_URL=http://codexsecure-api:8005
CODEXSECURE_SDK_API_TOKEN=<your-token>
CODEXSECURE_SDK_DEFAULT_ZONE=5
CODEXSECURE_SDK_TIMEOUT=5.0
CODEXSECURE_SDK_FAIL_OPEN=true
CODEXSECURE_SDK_ENFORCE_METHODS=false
```

**Status:** ⏳ Ready for execution

---

### Step 2: Start CodexHash with Docker Compose

**Action:** Launch CodexHash development environment

```bash
# From Docker/codexhash_api directory
docker-compose up -d
```

**Expected Behavior:**
1. CodexSecure API container starts (port 8005)
2. CodexHash API container starts (port 8001)
3. Healthcheck runs (30-second interval)

**Verify Startup:**
```bash
docker-compose logs codexhash-api | grep "CodexSecure SDK initialized"
```

**Status:** ⏳ Ready for execution

---

### Step 3: Test Zone Context Attachment

**Action:** Verify Zone 5 is attached to requests

```bash
# Test hash endpoint
curl -X POST http://localhost:8001/api/hash/compute \
  -H "Content-Type: application/json" \
  -d '{"text": "test-data", "algo": "sha256"}'
```

**Expected Response:**
```json
{
  "hash": "...",
  "algo": "sha256",
  "zone": "Z5"
}
```

**Status:** ⏳ Ready for execution

---

### Step 4: Run Validation Test Suite

**Action:** Execute CodexSecure integration tests

```bash
# From Docker/codexhash_api directory
pytest tests/test_codexsecure_integration.py -v
```

**Expected Result:**
```
test_codexsecure_integration.py::TestSDKInitialization::test_sdk_enabled PASSED
test_codexsecure_integration.py::TestSDKInitialization::test_sdk_disabled PASSED
test_codexsecure_integration.py::TestZoneContext::test_zone_context_attached_to_request PASSED
... (22 more tests)

======================== 25 passed in 2.34s ========================
```

**Status:** ⏳ Ready for execution

---

### Step 5: Verify Method Validation (Optional)

**Action:** Test optional method policy enforcement

```bash
# Enable method validation
export CODEXSECURE_SDK_ENFORCE_METHODS=true

# Restart CodexHash
docker-compose restart codexhash-api

# Try DELETE (should be restricted in Zone 5)
curl -X DELETE http://localhost:8001/api/hash/compute
```

**Expected:**
- If allowed: 200 OK
- If restricted: 403 Forbidden (zone_not_allowed_in_zone)

**Status:** ⏳ Optional (advanced testing)

---

### Step 6: Test Fail-Open Behavior

**Action:** Verify requests proceed when CodexSecure is unreachable

```bash
# Stop CodexSecure API
docker-compose stop codexsecure-api

# Try hash endpoint (should still work with FAIL_OPEN=true)
curl -X POST http://localhost:8001/api/hash/compute \
  -H "Content-Type: application/json" \
  -d '{"text": "test", "algo": "sha256"}'
```

**Expected:**
- Response: 200 OK (fail-open allows request)
- No 503 Service Unavailable errors

**Note:** Only works if `CODEXSECURE_SDK_FAIL_OPEN=true`

**Status:** ⏳ Ready for execution

---

### Step 7: CodexTime Implementation

**Action:** Replicate CodexHash pattern for CodexTime

**Files Updated:**
- `Docker/codextime_api/docker-compose.yml` ✅
- `Docker/codextime_api/docker-compose.prod.yml` ✅
- `Docker/codextime_api/.env.example` — Needs CodexSecure SDK vars

**Next:** Add CodexSecure SDK client to CodexTime (Python SDK same as CodexHash)

**Update (2026-04-22):** Implemented in the CodexWeb3 FastAPI service at `Docker/CodexWeb3/codextime_api/src/main.py`:
- SDK init during app lifespan (`CodexSecureSDKClient.from_env(service_name="codextime")` + health probe)
- Zone headers attached to responses when SDK enabled
- Optional method enforcement (guarded by `CODEXSECURE_SDK_ENFORCE_METHODS=true`) for `/api/time/*`
- Diagnostic endpoint: `GET /api/codexsecure/status`

**Status:** ✅ Implemented (validation pending)

---

### Step 8: End-to-End Integration Test

**Action:** Full workflow test across both services

```bash
# Start both services
docker-compose -f Docker/codexhash_api/docker-compose.yml up -d
docker-compose -f Docker/codextime_api/docker-compose.yml up -d

# Test CodexHash (Zone 5)
curl http://localhost:8001/health

# Test CodexTime (Zone 3)
curl http://localhost:8007/health

# Verify both report Zone context
curl http://localhost:8001/api/debug/zone  # Should return Z5
curl http://localhost:8007/api/debug/zone  # Should return Z3
```

**Status:** ⏳ Ready for execution

---

## Files Modified in Phase 4

| File | Change | Status |
|------|--------|--------|
| `Docker/codexhash_api/docker-compose.yml` | Created (new dev compose) | ✅ |
| `Docker/codexhash_api/tests/test_codexsecure_integration.py` | Created (25 tests) | ✅ |
| `Docker/codextime_api/docker-compose.yml` | Updated (added CodexSecure) | ✅ |
| `Docker/codextime_api/docker-compose.prod.yml` | Updated (added CodexSecure) | ✅ |
| `tasks/task_2026-04-21.md` | Updated (Phase 4 progress) | ⏳ Pending |

---

## Approval Gates

### Before Proceeding to Phase 5 (Production):

- [ ] All 25 validation tests pass
- [ ] CodexHash health check passes (30s interval success)
- [ ] Zone context verified in requests
- [ ] Fail-open behavior confirmed
- [ ] CodexTime SDK client created and tested
- [ ] Both services pass validation test matrix

---

## Known Issues & Resolutions

| Issue | Status | Resolution |
|-------|--------|-----------|
| CodexSecure API image not available | ⏳ PENDING | Use local CodexSecure build or specify registry |
| Method validation endpoint missing | ⏳ INFO | Only needed if `ENFORCE_METHODS=true` |
| CodexTime `.env` missing SDK vars | ⏳ TODO | Add to CodexTime `.env.example` |

---

## Rollback Procedure (If Issues)

1. **Disable SDK:** Set `CODEXSECURE_SDK_ENABLED=false`
2. **Stop Services:** `docker-compose down`
3. **Remove SDK Vars:** Comment out CodexSecure environment in compose
4. **Restart:** `docker-compose up`
5. **Verify:** Services run without zone enforcement

---

## Success Criteria

✅ **Phase 4 Completion Defined As:**
1. Validation test suite created and documented
2. Docker Compose files updated for both services
3. Environment configurations validated
4. Zone assignments verified (Z5, Z3)
5. All files committed to version control
6. Documentation updated in task file

---

## Transition to Phase 5

**When Phase 4 is complete:**
- [ ] All tests pass locally
- [ ] Docker images build successfully
- [ ] Compose files validated with `docker-compose config`
- [ ] Next: Phase 5 (Production Validation & Rollout)

---

## References

- **Setup Guide:** `_docs/CodexSecure/SETUP_STANDARDIZATION.md`
- **Setup Plans:** `tasks/task_2026-04-21.md` (Phase 2 & 3)
- **Test Suite:** `Docker/codexhash_api/tests/test_codexsecure_integration.py`
