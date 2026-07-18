---
Date: 2026-07-18
Change: pickup-confirmation
Status: REVIEWED
HAS_UI_SURFACE: no
---

# pickup-confirmation Requirements

Distilled from the approved design: [2026-07-18-pickup-confirmation-design.md](2026-07-18-pickup-confirmation-design.md).

## Goals

- Persist a dispatch record in Redis when the driver service dispatches a driver: `dispatch:<requestID>` → `{sessionID, driverID, eta, status: "dispatched", routingKey}`
- Expose a new business endpoint on the driver service's existing HTTP server (:8082): `POST /dispatches/{requestID}/arrived`
- On arrival: transition record status `dispatched → arrived` and store notification "Driver <id> arrived at pickup" via the existing notification handler
- Surface the arrival in the frontend through the existing notification polling — zero frontend code changes
- Demonstrate cross-service validation: the chain `HTTP POST → driver → Redis → frontend polling` is validated by a Signadot sandbox plan (`pickup-confirmation-arrival`), not unit tests alone

## Non-Goals

- No frontend changes (UI already polls notifications)
- No driver status lifecycle beyond `dispatched → arrived` (that is Feature 1, driver-status)
- No auth on the arrival endpoint (demo app)
- No new Redis/config plumbing — reuse `pkg/config` Redis settings and the existing notification pattern
- No signadot-validate CLI integration beyond manual/scripted equivalent (CLI contract pending Joe's answers)

## Constraints

- Dispatch records live in the same Redis instance as notifications; TTL follows the notification pattern (30s SetEx) unless apply proves a longer window necessary
- The arrival notification must reconstruct `NotificationContext` from the persisted record — the record MUST store sessionID and routingKey (confirmed: `notifications.Store` requires them)
- Driver HTTP surface is `services/driver/processor.go` (health-only today); the new route registers on the same `:8082` mux — no second server
- k8s: driver has a Deployment but NO Service today (8082 is probe-only) — a Service (and containerPort) must be added so the endpoint is reachable in-cluster for sandbox validation
- Idempotency: repeated `POST .../arrived` returns 200 and MUST NOT store a duplicate notification (notification handler already dedupes by ID; use a deterministic notification ID, e.g. `req-<id>-arrived`)
- Go 1.x, `go test ./...` green from repo root

## Success Criteria

- Unit: dispatchstore CRUD round-trip; arrived handler returns 404 (unknown id), 200 idempotent (already arrived, no duplicate notification), 200 transition (status updated + notification stored)
- Cluster: in a Signadot sandbox running the forked driver against baseline redis/frontend, `POST /dispatches/<id>/arrived` for a real dispatched request causes frontend `GET /notifications` to include "Driver X arrived at pickup"
- `go test ./...` passes; baseline behavior (dispatch flow, existing notifications) unchanged when the new endpoint is unused

## User Stories

- As a rider, I want to be notified when my driver arrives at pickup so that I know to head out
- As a demo operator (Signadot), I want a cross-service behavior that unit tests cannot cover so that sandbox validation demonstrably adds value
- As a developer, I want dispatch state persisted with validation (404 on unknown dispatch) so that the arrival API is a real state machine, and the follow-up driver-status change can reuse the store

## Open Questions

(none — design APPROVED; decisions 1-4 recorded in the design doc)

## Referenced Capabilities

- ADD `dispatch-tracking` — new capability: dispatch record persistence + arrival transition + arrival notification (services/driver: dispatchstore.go NEW, consumer.go write-at-dispatch, processor.go arrived route; k8s/base/driver.yaml Service)
