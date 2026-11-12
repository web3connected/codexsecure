# Discovery coverage implementation

Implemented and verified October 21, 2026, in the CodexSecure Go API only. The supplied [Go contract](CODEXSECURE_DISCOVERY_COVERAGE_GO_CONTRACT.md) is the implementation specification. These changes require the coverage migration before this API version is deployed; no migration or deployment was performed against live services.

1. **Files changed**

   Runtime:
   - `internal/communications/discovery.go`
   - `internal/communications/batches.go` (new)
   - `internal/communications/postgres.go`
   - `internal/communications/assessment.go`
   - `internal/communications/store.go`
   - `internal/communications/errors.go`
   - `internal/handlers/communications.go`
   - `internal/middleware/communication_auth.go`

   Verification:
   - `internal/communications/coverage_integration_test.go` (new)
   - `internal/communications/testdata/legacy_replay.sql` (new)
   - `internal/communications/authority_discovery_assessment_integration_test.go`
   - `internal/communications/database_contract_integration_test.go`
   - `internal/communications/postgres_integration_test.go`
   - `internal/communications/discovery_test.go`
   - `internal/communications/errors_test.go`
   - `internal/handlers/communications_integration_test.go`
   - `internal/handlers/communications_test.go`
   - `scripts/test_communication_contract.sh`

   Documentation: this report, the supplied contract copied into `docs/CODEXSECURE_DISCOVERY_COVERAGE_GO_CONTRACT.md`, and follow-up notices in `COMMUNICATION_CONFIGURATION_IMPLEMENTATION.md` and `AUTHORITY_DISCOVERY_ASSESSMENT_INTEGRATION.md`.

2. **Database contract alignment**

   Discovery writes `observed_type`, destination, method, and communication zone as independent evidence. Registry validation remains; the obsolete baseline-type and source-page-zone rejection paths are removed. Declaration item syntax remains validated; observed type no longer makes an unchanged declaration structurally invalid. Changes to the declaration itself still advance structural revision. The registered UUID, ownership, immutable first-seen baseline, approved configuration and approval history retain their existing protections. The new ownership constraint maps to `REGISTRATION_IDENTITY_CONFLICT`; unrelated database errors remain safely mapped.

3. **Discovery batch endpoints**

   Added `POST /sites/{site}/communications/discovery/batches`, `POST .../batches/{batch}/failures`, and `POST .../batches/{batch}/finalize` under the existing authenticated API group. They use strict body decoding, the existing UUID Idempotency-Key requirement, trusted `DiscoverySites` grants, and the three SECURITY DEFINER functions. Start/finalize actors come from the verified principal. Failure attempts follow the database ledger's retry semantics; start/finalize replay binding remains database-authoritative. No direct writes to protected coverage tables were introduced.

   Observation ingestion accepts optional `{batch:{key,sequence}}`; normalized batch identity and sequence enter the existing request fingerprint. The batch lookup is restricted to the site and its ID/sequence are inserted in the observation transaction. Legacy out-of-batch discovery remains supported and the database invalidates prior coverage.

   After a valid batch reference is parsed, ingestion failures roll back before a separate transaction records `rejected` or `failed` with the original API code. A bounded context independent of caller cancellation allows recording a processing failure after cancellation. If recording also fails, the original status/code is preserved and `errors["batch.failure_recording"]` explicitly reports the missing ledger write. Database details are not exposed.

4. **Current observation evaluation**

   Assessment evidence comes exclusively from `communication_current_observations`. Every instance in the complete generation is evaluated individually, including multiple instances with the same browser identifier. Configuration display prefers all current generation instances and falls back to the newest instance per object/page/browser identifier only on pages without current coverage. Both paths read `observed_type`.

   Independent method/destination/type/zone mismatches yield Unauthorized without changing approval. Matching current evidence restores Approved. Existing structural invalidation and corrected-structure reapproval behavior remains covered by regression tests.

5. **Assessment completeness**

   Coverage checks, registration enumeration, absence reads, and evaluation share one read-only repeatable-read transaction. Responses use `coverage_basis: "finalized_generation"` and additive `coverage:{generation,batch_key,coverage_at}` metadata when a generation exists.

   Page coverage requires a current complete, uninvalidated latest generation. Site coverage additionally checks the latest site generation, every manifest page's current child, and all registered pages against the manifest. All scoped registrations must be accounted for by current observations or current absence records and evaluated successfully. Missing generation, open/incomplete/failed/stale generation, invalidation, missing pages, and missing registration evidence prevent completeness. Database completeness issue codes and existing progress information are preserved. Absences are findings in `coverage_gaps`, without adding completeness failures. Arbitrary historical evidence cannot establish completeness. Processing failures remain non-2xx with incomplete assessment data.

6. **SQLSTATE and replay compatibility**

   Added message-sensitive mappings for `DISCOVERY_BATCH_NOT_FOUND`, `DISCOVERY_BATCH_CLOSED`, and `REPLAY_CONFLICT`, preserving existing `STALE_REVISION`, `ALREADY_PROCESSED`, and `CONFIGURATION_INVALID`. Decision functions continue receiving the verified principal actor and retain their request/response signatures. Go does not assign draft versions or implement a second canonical digest algorithm: database lifetime generations and request digest verification are authoritative. The existing event-existence lookup only bypasses redundant draft validation for a possible replay; the database still validates the entire replay request.

7. **Verification**

   Passed:
   - Full Go regression: `go test ./...`.
   - Full race-enabled regression with integration build tags: `go test -race -tags integration ./...`.
   - Final communications/HTTP race-enabled run after added ownership and concurrency cases: `go test -race -tags integration ./internal/communications ./internal/handlers`.
   - Disposable bootstrap script syntax check.

   PostgreSQL 16 ran in a newly initialized disposable cluster at `127.0.0.1:55439`, database `codexsecure_configuration_test`. The bootstrap uses the complete SQL equivalent of the first PageMeta Blueprint migration and the unchanged registry, configuration, and coverage SQL migration sources. A fixture generated by the old decision function before migration four verifies legacy replay-key rejection. Runtime-role connections exercise the actual grants, triggers, views, and SECURITY DEFINER functions. Other database integration tests requiring separately supplied DB_* credentials skip normally; no production credentials were loaded.

   Coverage includes changed intent and restoration, stable UUID/policy, independent observed type, Page Object/Universal isolation, HTTP batch lifecycle, identical starts/finalizations, gaps, duplicate sequences, durable validation/processing failures, corrected retries, failure-recording failure, stale and invalidated generations, empty complete pages, site manifests, absences, complete/incomplete assessments, draft ABA, changed request/actor replay conflicts, legacy keys, strict decoding, trusted site grants, and repeated races of two concurrent observations against finalization.

   Logs for this session:
   - `/tmp/codexsecure-coverage-regression.log`
   - `/tmp/codexsecure-coverage-integration.log`
   - `/tmp/codexsecure-coverage-final-tests.log`
   - `/tmp/codexsecure-coverage-final-integration.log`

8. **Remaining external requirements**

   The Web3Codex owner must apply the reviewed coverage migration through its own authorized release process before deploying this Go API version. Gateway site binding, administrator assertion verification/key registration, installation signing, and transparent status/envelope forwarding remain separate work. Existing production authorization continues to fail closed without a verified principal; identity handling was not relaxed. No live configuration or SDK feature flag was changed.

9. **Scope confirmation**

   All source and documentation edits are within this Go API project. Sibling Web3Codex migration sources were read only. Database writes were confined to newly created disposable PostgreSQL clusters. No live database, sibling project, gateway, credential, environment file, or production configuration was modified. No commit, push, deployment, or enforcement activation occurred. Disposable servers were stopped after verification.
