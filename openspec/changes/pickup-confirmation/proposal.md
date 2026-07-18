---
Date: 2026-07-18
Change: pickup-confirmation
HAS_UI_SURFACE: no
Requirements: docs/superpowers/specs/2026-07-18-pickup-confirmation-requirements.md
---

## Why

HotROD's ride flow ends at "Driver X arriving in Ys" — the rider never learns the driver actually arrived, and the driver service holds no dispatch state to act on. Adding a real arrival confirmation creates the simplest cross-service behavior that unit tests cannot cover, which is exactly what the Signadot sandbox-validation integration needs as its first demo.

## What Changes

- Driver service persists a dispatch record in Redis at dispatch time: `dispatch:<requestID>` → `{sessionID, driverID, eta, status: "dispatched", routingKey}`
- New business endpoint on the driver service's existing `:8082` HTTP server: `POST /dispatches/{requestID}/arrived` (404 unknown id; idempotent 200 when already arrived; otherwise transitions status and stores "Driver X arrived at pickup" notification)
- k8s: add a Service for the driver Deployment (8082 currently probe-only) so the endpoint is reachable in-cluster
- Frontend: no code change — existing notification polling surfaces the arrival

## Capabilities

### New Capabilities

- `dispatch-tracking` — dispatch record persistence, arrival transition, arrival notification

### Modified Capabilities

(none — openspec/specs/ is empty; this is the repo's first change)

## Impact

- `services/driver/dispatchstore.go` — NEW: Redis-backed dispatch record CRUD
- `services/driver/consumer.go` — write dispatch record at end of `processDispatchRequest`
- `services/driver/processor.go` — register `POST /dispatches/{id}/arrived` on the existing `:8082` mux
- `k8s/base/driver.yaml` — containerPort + new Service for 8082
- `pkg/config` — reused as-is (Redis addr/password); no new config
- Signadot: sandbox plan `pickup-confirmation-arrival` validates `HTTP POST → driver → Redis → frontend polling` end-to-end

## Out of Scope

- Driver status lifecycle beyond `dispatched → arrived` (deferred to `driver-status`)
- Frontend UI changes of any kind
- Auth/rate-limiting on the arrival endpoint (demo app)
- `signadot-validate` CLI integration (manual/scripted equivalent this pass; CLI contract pending)
