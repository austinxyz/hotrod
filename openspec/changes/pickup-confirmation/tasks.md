## 1. Dispatch store (Redis-backed record CRUD)

### Contract
- **Spec**: "When the driver service completes dispatch processing (best-ETA driver selected), it SHALL persist a dispatch record in Redis under key `dispatch:<requestID>` containing sessionID, driverID, eta, routingKey, and status `dispatched`." (store layer: schema + CRUD; the consumer write itself is group 2)
- **Runtime**: `go test ./services/driver/...` → expected: all dispatchstore tests pass (round-trip, missing-key, status update), no build errors
- **Code**: separate `dispatchstore.go` component with small interface (Store/Get/UpdateStatus), own Redis client via pkg/config, mirrors pkg/notifications handler pattern; record includes routingKey + sessionID (NotificationContext reconstruction); TTL 5 min SetEx constant; tests use github.com/alicebob/miniredis/v2 (new test-only dep)
- **Threshold**: 80

- [ ] 1.0 CONTRACT — write openspec/changes/pickup-confirmation/contracts/group-1.md with the ### Contract block above; confirm all three fields (Spec, Runtime, Code) are non-empty before proceeding
- [ ] 1.1 RED — failing test: DispatchStore round-trip — Store record for requestID R then Get returns same {sessionID, driverID, eta, routingKey, status: "dispatched"}; Get unknown ID returns not-found sentinel (miniredis)
- [ ] 1.2 GREEN — implement services/driver/dispatchstore.go: DispatchRecord struct (JSON), interface {Store, Get, UpdateStatus}, Redis client via pkg/config, 5-min SetEx TTL, otel span per op (follow pkg/notifications/handler.go pattern)
- [ ] 1.3 RED — failing test: UpdateStatus transitions "dispatched"→"arrived" and persists; UpdateStatus on missing key returns not-found
- [ ] 1.4 GREEN — implement UpdateStatus (GET + mutate + SET preserving TTL); pass 1.3
- [ ] 1.E EVAL — spawn evaluator subagent (haiku); reads contracts/group-1.md + spec + design + group diff; invokes superpowers:requesting-code-review (CRITICAL/HIGH = BLOCK); scores Spec/Runtime/Code; total ≥ 80 → PASS; < 80 → append FIX tasks + retry (max 3 attempts, plateau < 5pt = escalate)

## 2. Arrival flow (consumer write + HTTP endpoint + k8s Service)

### Contract
- **Spec**: "Record written after successful dispatch" (consumer); "The driver service SHALL expose `POST /dispatches/{requestID}/arrived` ... set status to `arrived` and store a notification 'Driver <driverID> arrived at pickup' ... respond 200"; "Unknown dispatch rejected" (404, no notification); "Repeated arrival calls SHALL respond 200 without storing a duplicate notification" (deterministic ID `req-<requestID>-arrived`); "The driver Deployment SHALL expose port 8082 via a k8s Service"; "arrival notification SHALL flow through the existing notification store so the frontend's /notifications polling surfaces it with zero frontend changes"
- **Runtime**: validated by signadot plan `pickup-confirmation-arrival`
- **Code**: handler registered on existing :8082 mux in processor.go (no second server); NotificationContext rebuilt from record fields only (no baggage on HTTP POST); GET+SET race accepted — deterministic notification ID dedupes; k8s Service added in k8s/base/driver.yaml so all overlays inherit
- **Threshold**: 80

- [ ] 2.0 CONTRACT — write openspec/changes/pickup-confirmation/contracts/group-2.md with the ### Contract block above
- [ ] 2.1 RED — failing test: arrived handler — POST unknown requestID → 404, no notification stored (mock/miniredis + recorded notification interface)
- [ ] 2.2 GREEN — implement arrived handler skeleton in processor.go: route `POST /dispatches/{id}/arrived` on the :8082 mux, 404 path via dispatchstore.Get
- [ ] 2.3 RED — failing test: successful transition — record in "dispatched" → 200, status becomes "arrived", notification "Driver <driverID> arrived at pickup" stored with ID `req-<requestID>-arrived` for the record's session/routingKey
- [ ] 2.4 GREEN — implement transition path: UpdateStatus + notifications.Store with NotificationContext rebuilt from record; pass 2.3
- [ ] 2.5 RED — failing test: idempotency — record already "arrived" → 200, Store called with same deterministic ID (handler-level; dedupe itself is existing notification behavior)
- [ ] 2.6 GREEN — implement idempotent branch; pass 2.5
- [ ] 2.7 RED — failing test: consumer writes dispatch record — after processDispatchRequest selects bestDriver, dispatchstore contains record for the request (refactor seam: inject dispatchstore into Consumer)
- [ ] 2.8 GREEN — wire dispatchstore into Consumer; write record at end of processDispatchRequest (after bestETA succeeds, alongside the dispatched notification); pass 2.7
- [ ] 2.9 GREEN — k8s: add containerPort 8082 + Service `driver` (port 8082) to k8s/base/driver.yaml; verify overlays build (`kubectl kustomize k8s/overlays/prod/devmesh | grep -A6 'name: driver'`)
<!-- This group's Contract Runtime binds a signadot plan: -->
- [ ] 2.V VALIDATE — bind params in openspec/changes/pickup-confirmation/signadot-plans/pickup-confirmation-arrival.yaml (URLs, requestID capture now known); run signadot-validate against the real cluster (austin-staging-1; manual/scripted equivalent until CLI contract confirmed); append the structured verdict to eval-log.md; any failed assertion = Runtime floored → treat as BLOCK
- [ ] 2.E EVAL — spawn evaluator subagent (haiku); reads contracts/group-2.md + spec + design + group diff; invokes superpowers:requesting-code-review (CRITICAL/HIGH = BLOCK); scores Spec/Runtime/Code; if a 2.V verdict exists in eval-log.md, Runtime score = that verdict (pass=100, fail=0), not subagent judgment; total ≥ 80 → PASS; < 80 → append FIX tasks + retry (max 3 attempts, plateau < 5pt = escalate)

## 3. Verification + ship

<!-- No Contract/EVAL block for this group — verification-and-ship groups run cross-cutting checks, not per-feature harness evaluation -->

- [ ] 3.1 Run backend test suite — `go test ./...` from repo root, no regressions (project.test_commands from openspec/config.yaml)
- [ ] 3.2 Baseline behavior check — existing dispatch flow unchanged when the new endpoint is unused (dispatch notifications still arrive; no errors in driver logs)
- [ ] 3.3 Run superpowers:verification-before-completion (run project.test_commands from openspec/config.yaml; run project.custom_verification_checks — none configured)
