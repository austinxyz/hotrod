# Capability Specs

Durable capability specifications for hotrod-opsx. Each capability folder holds
its `spec.md` (requirements + scenarios) and, when present, `plans/` — validated
Signadot plans registered at archive time (the versioned plan library; future
changes touching the same behavior reuse these instead of authoring from scratch).

## Capabilities

### `dispatch-tracking` ✅ implemented
**User story**: As a rider, I'm notified when my driver arrives at pickup.
**Introduced by**: pickup-confirmation (2026-07-18)
**Backend**: Redis dispatch records (`dispatch:<requestID>`, 5-min TTL); `POST /dispatches/{requestID}/arrived` on driver `:8082` (404 unknown / idempotent 200 / transition + notification); driver k8s Service
**Frontend**: none — arrival surfaces through the existing notification polling
**Validation**: `plans/pickup-confirmation-arrival.yaml` — Signadot plan, 5 assertions green on austin-staging-1 (sandboxed fork of driver against baseline)
