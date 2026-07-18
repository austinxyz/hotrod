## Context

HotROD driver service consumes dispatch requests from Kafka (`services/driver/consumer.go` → `processDispatchRequest`), computes the best-ETA driver, and stores "Driver X arriving in Y" notifications via `pkg/notifications` (Redis, 30s SetEx, ID-dedupe inside `Store`). Its HTTP surface (`services/driver/processor.go`) serves only `/healthz` on `:8082`; k8s exposes the driver as a Deployment with a probe on 8082 but no Service. Nothing persists dispatch state, so no external event can act on a dispatch after the fact.

Approved brainstorm design: [docs/superpowers/specs/2026-07-18-pickup-confirmation-design.md](../../../docs/superpowers/specs/2026-07-18-pickup-confirmation-design.md). This doc refines it against the actual code.

## Goals / Non-Goals

**Goals:**
- Persist dispatch records so arrival is a real, validatable state transition
- Arrival endpoint on the driver's existing HTTP server; notification through the existing store; zero frontend changes
- Endpoint reachable in-cluster → Signadot sandbox can validate the full chain

**Non-Goals:**
- Status lifecycle beyond `dispatched → arrived` (future `driver-status` change reuses the store)
- Auth, rate limiting, config plumbing, frontend work

## Decisions

1. **`dispatchstore.go` as a separate component with a small interface** (`Store/Get/UpdateStatus` or equivalent) — mirrors the existing `pkg/notifications` handler pattern (own Redis client via `pkg/config`, otel instrumented). Alternative rejected: folding CRUD into consumer.go (untestable, blocks driver-status reuse).
2. **Record schema includes `routingKey` + `sessionID`** — the arrival handler runs outside any Kafka context; it must reconstruct `NotificationContext` purely from the record. Confirmed requirement from `notifications.Store` signature. Alternative rejected: re-deriving context from baggage (no baggage on a plain HTTP POST).
3. **Deterministic notification ID `req-<requestID>-arrived`** — reuses `Store`'s existing dedupe loop for idempotency instead of new dedupe logic. Matches existing ID convention (`req-%d-finding-driver`, `req-%d-dispatched-driver`).
4. **Record TTL follows the notification pattern (SetEx)** but longer than 30s — a dispatch must outlive the demo interaction window so the manual `POST arrived` has something to hit; start with 5 minutes, tune at apply if the demo flow needs more. Alternative rejected: no TTL (stale keys accumulate in a demo cluster).
5. **Route registration on the existing `:8082` mux in processor.go** — no second server, no new port. Handler does: GET record → 404 if missing → idempotent 200 if `arrived` → else UpdateStatus + Store notification → 200.
6. **k8s: add Service `driver` (port 8082) + containerPort to the Deployment** in `k8s/base/driver.yaml` — base layer so every overlay (including devmesh) inherits it. Signadot sandbox forks the driver workload; baseline Service routes to it via the operator's usual mechanics.
7. **Status transition without WATCH transaction** — plain GET + SET race window is acceptable for a demo: worst case a concurrent duplicate arrival stores the same deterministic notification ID, which dedupes anyway. Alternative rejected: full optimistic locking (complexity with no observable behavior difference here).

## Risks / Trade-offs

- [Requests arriving before record write (consumer crash mid-dispatch)] → arrival returns 404; acceptable, spec'd
- [TTL expiry before demo POST] → 5-minute TTL, tunable constant in dispatchstore.go
- [Signadot sandbox routing to forked driver via Service] → devmesh overlay already routes baseline traffic; verify during N.V with the sandbox spec
- [Redis client duplication (notifications + dispatchstore each own a client)] → follows existing per-component pattern; consolidation is out of scope

## Migration Plan

Additive only: new key namespace `dispatch:*`, new endpoint, new Service. No schema/data migration; rollback = revert commits. Baseline behavior unchanged when the endpoint is unused.

## Open Questions

(none — remaining unknowns are execution-time: exact sandbox spec fields at N.V, TTL tuning)
