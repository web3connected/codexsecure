CodexSecure Architecture
Revision: 3 — First Draft for Review (Z0 named Boundary; alignment layers recorded)
Date: October 25, 2026
Status: Architecture alignment draft; not an implementation authorization

Product: CodexSecure (Web3Codex ecosystem)

1. Purpose and scope
CodexSecure is Web3Codex's relationship-based communication security product. It applies 12-zone structure and the Seven-Fold Process (7FP) to recognize, organize, and govern communications. It is developed alongside, and integrates with, the independent foundational products CodexHash and CodexTime. CodexSecure does not define their underlying mathematics or exist as their parent product.

This document supersedes the architectural narrative of the February 8, 2026 CodexSecure document where that narrative conflicts with current decisions. It does not assert that every described feature is implemented. The October 20, 2026 verified implementation note remains a baseline for what was actually observed.
Status vocabulary
- Verified: Observed in the October 20 implementation review or associated tests.
- Agreed architecture: Explicitly established product behavior, whether or not fully implemented.
- Pending validation: Requires source-code inspection, test evidence, or further design review.
2. Foundation: 12–60–7 and product boundaries
The 12–60–7 Framework is a mathematical framework grounded in relationships studied through the sexagesimal (base-60) system. The clock is an accessible representation: twelve positions, sixty divisions, cyclical motion, and return. Inversion Math is part of the broader mathematical work; it is not another name for the entire framework.
- CodexSecure — 12 and 7: twelve operational zones and the Seven-Fold Process for communication relationships and operations.
- CodexTime — 60: an independent time and synchronization technology, including the TIU Quantum Time model. Its mathematical foundation and synchronization mechanisms belong in CodexTime documentation.
- CodexHash — CodexHash: independent harmonic/quantum hashing technology based on Inversion Math. It is not assigned a forced numerical role in 12–60–7. CodexSecure may consume its hashing capabilities for structural identity.
- CodexNetworks: planned network infrastructure that extends zone separation to dedicated environments/subdomains, including an isolated public proxy for the Z0 Boundary.
- CodexMind: a distinct AI intelligence technology in the wider ecosystem, not a stage of CodexSecure.
No CodexSecure documentation should redefine CodexTime's TIU as “Trust Integrity Updates,” or describe CodexHash as having been designed merely to hash zone patterns.
3. Physical routing: the building model
Agreed architecture. The building analogy describes the physical and logical roles:
- Z0 is the Boundary: the building's entrance and the unassigned state. Every incoming communication starts there without operational-zone access. It is not one of the twelve zones.
- A route file is a door: its placement in the application’s zone route structure already identifies which operational zone the route belongs to. The zone is not guessed from an email field, URL keyword, or AI-generated label.
- MCP is the doorman: it performs pre-routing intent alignment at entry, using the route's known zone context and learned harmonization recognition.
- CodexSecure is the security system: it owns the zone-intent and approved communication relationship process, and continues the Seven-Fold Process.
- Harmonization is the doorman's memory: recognition improves as prior communications and their evaluated outcomes establish a history of relationships. Recognition is not the same thing as approval.
A known route destination is therefore not equivalent to an automatically approved communication. The route identifies the door; the request's intent and relationship must align with that door.
Example
A developer places an authentication route in the Z2 / Identity route folder. That is the route's established operational zone. An incoming communication first arrives at the Z0 Boundary. During Initialize, the MCP pre-routing component evaluates its intent against the known Z2 route context and harmonization history. An aligned communication can proceed with that Z2 context into CodexSecure's remaining processing. An unrecognized or unaligned communication stays at the Z0 boundary for the appropriate zone-intent handling; it does not acquire Z2 access merely by targeting the route.
4. Stage 1 — Initialize: ingress and MCP pre-routing
Agreed architecture. Initialize is the first stage of the Seven-Fold Process. Its ingress responsibility includes MCP pre-routing.

Three checks are involved, and they must not be described as one another:
- Physical alignment (Web3Codex): a route's zone comes from its zone directory, the physical identity dotfile in that directory, and the canonical registry. It is resolved at route registration and preserved in the route cache.
- Registry alignment (Web3Codex SDK ↔ Go): the SDK calls POST /secure/api/v1/policy/alignment to confirm both systems agree on the zone identities (and, separately, definitions). It compares registries; it never receives a route, method or communication. Details: docs/ZONE_ALIGNMENT.md.
- Request processing (Seven-Fold Process): progression through the seven stages. Stage 1 currently establishes physical and registry alignment.
Registry alignment is not per-request authorization.

Implemented today (Web3Codex SDK, 2026-10-25): Initialize checks the route's physical identity and the registry alignment result, stores both in the shared Seven-Fold context, and continues or stops stage progression on that result. Stage 1 does not deny the HTTP request. Steps 4–5 below (MCP intent evaluation and harmonization history) are not part of that implementation.
1. Receive the incoming request in the default Z0 context.
2. Resolve the targeted registered route and its physically established zone from the application route structure.
3. Preserve the information required for subsequent processing and correlation.
4. Evaluate the communication's intent alignment with the known route zone, informed by existing harmonization recognition/history.
5. Carry forward the pre-routing result and context to CodexSecure's zone-intent processing; if no alignment is established, the communication stays at the Z0 Boundary.
The MCP component is not an independent final authority that invents the route's zone. Its purpose here is pre-routing alignment and recognition. The authoritative treatment of communication relationships belongs to CodexSecure's processing.

Pending validation: the exact ingress handoff, MCP transport, persistence/feedback mechanism, and behavior for first-time unfamiliar but legitimate communications require inspection against the current implementation. Do not invent an automatic approval rule or an undocumented learning algorithm.
5. Harmonization: recognition over time
Agreed architecture. Harmonization is a relationship history, not a one-off classification. Like a doorman who initially does not recognize a visitor, MCP may have little prior familiarity with a new communication. Repeated evaluated interactions can improve its recognition of established communication patterns and their zone associations.

Keep these concepts separate:
- Route-zone assignment: established by physical route placement.
- Pre-routing intent alignment: MCP's Initialize-stage assessment of whether the communication fits that known door/zone.
- Communication recognition and approval: CodexSecure's approved Communication Object relationships.
- Structure-to-zone compatibility: whether the observed communication structure aligns with the zone.
- Harmonization history: learned recognition informed by prior outcomes, without replacing explicit approved relationships.
The February document's reinforcement / phase-lock language may remain as a conceptual historical reference, but its concrete reward/penalty mechanics and implementation status are pending validation. Do not present those mechanics as deployed unless verified.
6. Twelve operational zones and the Z0 Boundary
Agreed architecture. There are exactly twelve zones, Z1–Z12, each a class of internet traffic: Public, Identity, Operations, Protected, Processing, Logging, Communications, Payments, Analytics, Streaming, Synchronization, Administration. Route files are grouped by zone in the application. The configuration-backed registry (Go: policy.DefaultZones(); Web3Codex: codexsecure.zone_registry), rather than a table in this document, is the authority for canonical keys and rules.

Z0 — Boundary is the entry boundary and unassigned state: communication that has not been assigned to an operational zone or has not established a valid zone relationship. It is outside the zone structure and is not counted among the twelve zones. Go keeps the technical identifier Z0 (thirteen identifiers in total) for compatibility. Counts of zones mean Z1–Z12. Z0 policy is read-only with GET allowed. Z0 is not in itself a quarantine or a malicious-traffic destination: quarantine and rejection are separate security outcomes, decided by the processing stages.

Earlier names for Z0 ("Negative Zone", "Lobby / Gateway") are historical. Go still accepts the keys negative and lobby as compatibility aliases for Z0.

CodexNetworks (planned): the Boundary is intended to become an isolated public proxy with no trusted ties to critical infrastructure. The separation between the logical Z0 Boundary and its eventual network deployment must be explicit.

Fractal subzones may be described as an architectural concept, but specific subzone mappings and middleware inheritance should not be labeled verified without code/configuration evidence.
7. Page Object Identity and Communication Objects
Agreed architecture; implementation varies by component.
Page Object Identity / PageMeta
Page Object Identity represents the structural process behind a registered route, including route identity, effective HTTP method, controller/action, zone, and declared intent. It is broader than a page URL. The structural identity can use CodexHash without redefining CodexHash's independent purpose.

Page Intent is static HTML metadata for the page, distinct from authorization policy. Its agreed storage path is through the isolated codex_zones service via the existing SDK/API, with a 24-hour cache for that metadata.
Communication Objects
Communication Objects represent registered communication interfaces. Their identity and approved configuration cover the communication's method, destination, zone, and administratively assigned scope. A communication's observed submission structure (field names, nesting, types) can be fingerprinted independently from its data values. Changes in structure can require a new structural evaluation without erasing the Communication Object's identity and history.

Two relationships should be represented independently:
1. Communication Object ↔ site/application recognition and approval.
2. Communication structure ↔ zone compatibility/harmonization.
Effective HTTP method matters, including framework method overrides. Destructive operations such as DELETE have an agreed additional authorization design; exact signature lifecycle and rollout remain release work.
8. Seven-Fold Process: no invented stage map
Agreed architecture. CodexSecure applies the Seven-Fold Process. Initialize is Stage 1 and includes the MCP pre-routing responsibility described above.

The February diagram assigning Stage 1 to MCP, Stages 2–5 to generic Go operations, Stage 6 to zone alignment, and Stage 7 to “Execute/Record” is withdrawn as an authoritative 7FP definition. It was not grounded in the actual middleware architecture.

The Laravel SDK coordinates the process and invokes the CodexSecure Go API for responsibilities that belong there; 7FP is not seven separate network requests. The precise names, order, and ownership of Stages 2–7 must be copied from the authoritative current middleware/process definitions, not reconstructed from the old diagram.

Pending validation: authoritative 7FP stage list, per-stage SDK/API responsibilities, stop/continue behavior, and test coverage. Until reconciled, this document intentionally does not invent those six stage definitions.
9. Current service and database boundaries
Verified October 20, 2026:
- CodexSecure uses the isolated codex_zones service database.
- Laravel uses its separate application database and communicates through the SDK/API.
- Dedicated migration tooling is in the Web3Codex workspace using a separate schema-admin connection; Go does not run DDL at startup.
- The reviewed physical database contained pagemeta_packages and migration history.
- /health measures process liveness; /ready measures database connectivity.
- /api/v1/policy/zones exposes configuration-backed definitions for the Z0 Boundary and Z1–Z12.
- /api/v1/zones/health was an independent SQL-registry endpoint that failed because security_zones was absent; its behavior is not evidence of zone-harmonization health.
- PageMeta recording/recognition was implemented and tested. Learned harmonization and communication authorization were identified as separate development work.
The historical tables harmonic_zones, zone_patterns, zone_validations, resonance_logs, and tiu_events are not a current provisioning instruction. Do not create them merely because they appeared in the February design. Refer to docs/CodexSecure/CODEXSECURE_STABILITY_REVIEW.md for the original review evidence and follow-up proposals.
10. Integration boundaries
CodexSecure ↔ CodexHash: CodexSecure may use CodexHash for fingerprints and structural identities; CodexHash remains an independent Web3 hashing product.

CodexSecure ↔ CodexTime: CodexSecure may use CodexTime's timing and synchronization capabilities; TIU and the time model belong to CodexTime. Do not substitute a generic service heartbeat for CodexTime's quantum-time model.

CodexSecure ↔ CodexNetworks: logical zone routing and relationship governance belong to CodexSecure; the proposed isolation of zones into network environments belongs to CodexNetworks.

CodexSecure ↔ CodexMind: CodexMind is a separate intelligence product, not the default authority for CodexSecure's communication approvals or zone routing.
11. Historical sections replaced or deferred
The following February 2026 descriptions are not authoritative current architecture:
- MCP guessing zone from arbitrary URL/payload heuristics as if physical route-zone placement were unknown.
- A generic seven-stage diagram that does not match the current 7FP.
- SHA-256 pattern hashes as the defining architecture for CodexSecure, or as a definition of CodexHash.
- “TIU = Trust Integrity Updates.”
- Historical database schema lists as instructions to provision current tables.
- MCP machine-learning/threat-score claims without current implementation evidence.
- Historical API lists and subzone middleware examples without verification.
- Performance figures such as under-50-ms overhead as if they were published benchmarks; these were targets only.
12. Open review items before approval
1. Confirm authoritative Stages 2–7 from current SDK middleware and Go integration; document names, responsibilities, and stop points.
2. Verify the MCP ingress/pre-routing physical implementation and the handoff from the Z0 Boundary to known zone context.
3. Define how harmonization memory is updated by evaluated outcomes, without treating familiarity as authorization.
4. Confirm handling of first-time legitimate communications whose intent aligns but have no recognition history.
5. Reconcile the exact current zone configuration and any subzone implementation.
6. Confirm CodexHash and CodexTime integration points against their own product documentation.
7. Reconcile PageMeta scheme-2 structural identity, Communication Object registration, structure-to-zone harmonization, and destructive-operation authorization against the current architecture document.
8. Verify service endpoints and database migrations before describing them as release-ready.
Review principle: Preserve product boundaries, established architectural decisions, and verified implementation facts. Do not fill missing definitions with invented process logic. This is a first draft for review, not approval to modify code or provision databases.