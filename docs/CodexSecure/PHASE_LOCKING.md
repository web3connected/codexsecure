# CodexSecure Phase Locking

**Last Updated:** April 27, 2026  
**Related:** `ROUTE_DRIVEN_PAGE_SCHEMA_ARCHITECTURE.md`, `ARCHITECTURE.md` (codexsecure-api)

---

## What a Phase Is

A **phase** is the declared identity of a route.

It is not assigned at runtime. It is read from the naming conventions that programmers already use when they build a route. By the time the first request arrives, CodexSecure already knows what the route is supposed to be.

The login route is the clearest example:

| Signal Source | Value |
|---|---|
| Controller class | `LoginController` |
| Model binding | `LoginModel` |
| Route name | `login.submit` |
| URI | `/login` |
| View template | `auth.login` |
| Declared form fields | `email`, `password` |

All six signals say the same thing: this is a credential-submission route in the identity domain.

That agreement is the phase. CodexSecure reads it once and locks it.

---

## Intent Sources

CodexSecure reads phase intent from six sources. All are required to converge before a phase is locked.

### Controller

The controller class name declares the action domain. `LoginController` signals identity/auth intent. `PaymentController` signals payment intent. The class name is not ambiguous — it is the programmer's primary intent declaration.

### Model

The model binding confirms the expected data shape. `LoginModel` expects credential fields, not profile fields, not payment fields. If the model does not match the controller intent, that is a schema inconsistency that needs resolution before locking.

### Route Name

Dot-notation names group intent families. `login.submit` belongs to the `login.*` family. All members of a family share the same phase anchor. A family can have multiple routes (e.g., `login.show`, `login.submit`, `login.resend`) while maintaining the same phase identity.

### URI

The URI path provides a keyword match. `/login` matches the `identity` keyword strategy in CodexRouteAnalyzer. The route name and the URI should agree. If they do not, CodexSecure flags the discrepancy during schema generation.

### View

The view template name confirms render intent. `auth.login` declares that this route renders an authentication login surface. A structural hash of the rendered template is stored as a **phase anchor** — a fingerprint of what the page looked like when the phase was locked.

### Form Fields

The declared input keys are locked as the field contract. For the login route: `email` and `password`. Only those two keys are valid. Any additional key, renamed key, or missing required key is out-of-phase.

---

## The Locked Pattern

Once all six intent sources converge, CodexSecure locks a **pattern baseline** for the route.

The locked baseline stores:

- `intent` — human-readable label (e.g., `authenticate_user`)
- `allowed_keys` — exhaustive list of valid input field names
- `required_keys` — keys that must be present
- `field_patterns` — per-field data format hashes (`EMAIL_FORMAT`, `PASSWORD_FORMAT`, etc.)
- `max_fields` — hard ceiling on field count
- `allow_extra` — always `false` for a locked phase
- `view_structure_hash` — SHA-3 fingerprint of the view template at lock time

```json
{
  "route_name": "login.submit",
  "intent": "authenticate_user",
  "mutation_class": "credential_submission",
  "allowed_keys":   ["email", "password"],
  "required_keys":  ["email", "password"],
  "field_patterns": {
    "email":    "EMAIL_FORMAT",
    "password": "PASSWORD_FORMAT"
  },
  "max_fields":          2,
  "allow_extra":         false,
  "view_structure_hash": "sha3-512:a94f3c7e..."
}
```

The field patterns store a format descriptor (e.g., `EMAIL_FORMAT`), not the raw value. CodexSecure hashes the pattern structure, not the data. This means it can verify that an email field carries email-shaped data without ever reading or storing the email address.

---

## In-Phase vs Out-of-Phase

### In-Phase

A request is in-phase when:

- all required keys are present
- no extra keys appear
- each field value matches the locked format pattern
- total field count is within `max_fields`

```
POST /login
{ "email": "user@example.com", "password": "••••••••" }

✓ 2 fields — within max_fields: 2
✓ email → EMAIL_FORMAT — pattern matches
✓ password → PASSWORD_FORMAT — pattern matches
✓ no extra keys
→ in-phase → confidence score updated
```

### Out-of-Phase

A request is out-of-phase when any of the following is true:

1. **Extra field** — a key not in `allowed_keys` appears
2. **Wrong data format** — a field value does not match its locked pattern hash
3. **Renamed key** — an expected key is missing, replaced with a different name
4. **Injected key** — an entirely unknown key appears

```
// Extra field — privilege escalation attempt
POST /login
{ "email": "user@example.com", "password": "••••••••", "admin": true }
✗ "admin" not in allowed_keys → out-of-phase

// Wrong data format
POST /login
{ "email": "not-an-email", "password": "••••••••" }
✗ "email" does not match EMAIL_FORMAT → out-of-phase

// Renamed / injected keys
POST /login
{ "user_email": "user@example.com", "pass": "••••••••", "role": "superadmin" }
✗ none of these keys are in the locked pattern → out-of-phase
```

---

## What Happens to Out-of-Phase Data

Out-of-phase requests do not flow to the application.

The handling sequence:

1. Pattern mismatch detected during Stage 2–4 processing
2. Request reclassified as **non-phasing data**
3. Routed to Z0 (orchestrator / quarantine gate)
4. Stage 7 creates a quarantine record with:
   - `data_class: non_phasing`
   - `review_state: pending`
   - `phase_lock_effect: none`
5. Confidence score for the locked phase is **not updated**
6. Request does not proceed to application

The quarantine record captures the structural pattern of what was received (not the raw data) and the specific mismatch reason. This record is available for operator review, classification, or escalation.

---

## Score Protection

The confidence score for a locked phase can only be updated by in-phase (phasing) traffic.

Non-phasing requests produce quarantine records but have zero effect on the lock confidence. This means:

- A flood of extra-field injection attempts cannot weaken the lock
- Renamed-key attacks do not register as legitimate variance
- Malformed format data does not pull the format pattern in a new direction

The lock stays at its accumulated confidence regardless of the volume of non-phasing traffic routed through the same path.

---

## Dephasing

A route enters a **dephasing** state when sustained out-of-phase traffic is detected against it.

Dephasing does not alter the locked baseline. It triggers:

- increased quarantine logging volume
- escalation of `review_state` flags
- optional operator alert

The distinction: dephasing is an observation, not a change to the phase. The lock is not relaxed or re-derived during dephasing.

---

## Reinforcement Loop (MCP Gateway)

The Stage 1 MCP Gateway proposes a zone hypothesis for each request. Stage 6 makes the final zone decision.

When Stage 6 confirms the hypothesis, the hypothesis is **reinforced**. When Stage 6 rejects it, the hypothesis is **penalized**. Over repeated requests, this feedback loop moves gateway accuracy toward phase lock — where the gateway's hypothesis consistently matches the authoritative zone outcome.

Phase lock in the reinforcement sense is a convergence state, not a fixed configuration. A route reaches phase lock when the confidence that the Stage 1 hypothesis will match Stage 6's decision is high and stable.

---

## Phase Lock and the Page Schema

Phase locking depends on the `page-schema.json` artifact generated by the application (see `PAGE_SCHEMA_ARTIFACT_SPEC.md` and `ROUTE_DRIVEN_PAGE_SCHEMA_ARCHITECTURE.md`).

The schema artifact provides the input that CodexSecure uses to derive the initial locked baseline. Without a page schema, CodexSecure must infer intent from runtime signals only, which delays lock convergence.

With a schema:

- intent sources are declared before the first request
- the locked baseline is computed at schema load time
- phase lock starts from a strong prior, not a cold start

---

## Relation to Other Docs

| Document | How It Relates |
|---|---|
| `ROUTE_DRIVEN_PAGE_SCHEMA_ARCHITECTURE.md` | Defines how routes and pages declare their intent — the inputs to phase locking |
| `PAGE_SCHEMA_ARTIFACT_SPEC.md` | The file format CodexSecure loads to derive the initial locked baseline |
| `HANDLER_ZONE_OWNERSHIP_AUDIT.md` | Classifies which handlers are zone-canonical vs cross-cutting — affects which routes get phase-locked under a zone authority |
| `ZONED_ENDPOINT_ROUTE_PLAN.md` | The zoned route surface — each endpoint here will eventually get its own phase lock |
| `SETUP_STANDARDIZATION.md` | SDK integration pattern — SDK clients send zone headers that feed the reinforcement loop |
| `codexsecure-api ARCHITECTURE.md` | Stage-by-stage pipeline; Stage 2–4 perform the pattern detection that drives in-phase vs out-of-phase classification |
