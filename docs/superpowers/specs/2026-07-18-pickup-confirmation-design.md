# Design: HotROD Pickup Confirmation (hotrod-opsx)

**Status:** APPROVED
**Date:** 2026-07-18
**Home:** this repo (hotrod-opsx). Authored during brainstorm in opsx-superpowers on 2026-07-18.

## Context

First real-world example of the OpenSpec + Superpowers workflow applied to a Signadot project ([signadot/hotrod](https://github.com/signadot/hotrod)), per the collaboration with Signadot (Ani/Joe). Baseline hotrod is already deployed on the local kind cluster (`austin-staging-1`, connected to Signadot).

Feature chosen: **Pickup Confirmation** (Option 2 from the service-map analysis) — simplest cross-service behavior that unit tests cannot cover, demonstrating the value of Signadot sandbox validation.

## Project setup (5 steps)

1. **Fork + new session** — GitHub fork `signadot/hotrod` → `austinxyz/hotrod`; clone to `C:\Users\lorra\projects\hotrod-opsx`; work continues in a new Claude session rooted there. Original `C:\Users\lorra\projects\hotrod` clone stays as the baseline deployment source.
2. **Install opsx-superpowers skills** in the new project: `openspec/` (superpowers-driven schema), `.claude/commands/opsx/`, `docs/superpowers/specs/`, `CLAUDE.md`
3. **Extend the opsx skills for Signadot plan/validation** per [signadot-integration-spec.md](../../signadot-integration-spec.md): `signadot-plans/` dir at propose, `Runtime: validated by signadot plan <id>` Contract binding, `N.V VALIDATE` step between GREEN and EVAL, verdict → EVAL Runtime evidence
4. **Save both feature specs to the forked repo** — this doc (Feature 2: Pickup Confirmation, approved) and a Feature 1 (Driver Status Tracking) spec, which needs its own brainstorm/design pass before development
5. **Develop Feature 2 first** via `/opsx:explore → propose → apply → archive`, then Feature 1 (reusing `dispatchstore`)

## User-visible behavior

1. User requests a ride in the frontend (existing flow: frontend → Kafka → driver)
2. After the driver service computes best ETA, it **persists a dispatch record** in Redis:
   `dispatch:<requestID>` → `{sessionID, driverID, eta, status: "dispatched"}`
3. **New driver HTTP API**: `POST /dispatches/{requestID}/arrived`
   - record not found → 404
   - status already `arrived` → idempotent 200, no duplicate notification
   - otherwise: update status → `arrived`, store notification "Driver X arrived at pickup"
4. Frontend's existing notification polling displays the arrival — **zero frontend changes**

## Component changes

| Component | Change |
|---|---|
| `services/driver/dispatchstore.go` | NEW — Redis-backed dispatch record CRUD |
| `services/driver/consumer.go` | write dispatch record at end of `processDispatchRequest` |
| `services/driver/server.go` | add `POST /dispatches/{id}/arrived` to the existing driver HTTP server |
| `pkg/config` | reuse existing Redis config (no new config) |
| frontend | none |

## Decisions (with alternatives considered)

1. **Real state change over simulation** — an external API triggers arrival, not a timer. Rejected: auto-simulated arrival (no external surface to validate).
2. **API on driver service, not frontend** — driver state belongs to the driver domain. Rejected: frontend writing Redis directly (bypasses the cross-service boundary the demo exists to show).
3. **Stateful dispatch records over stateless notify-passthrough** — records enable validation (404 on unknown dispatch), idempotency, and lay the foundation for Option 1 (Driver Status Tracking) as the next change. Rejected: passing sessionID directly (driver can't validate anything).
4. **Fork-based repo** — pushable, reviewable by Joe, PR-able back to signadot/hotrod. Rejected: working in the upstream clone (can't push), fresh project wrapping hotrod (deploy/test complexity).

## Why this is a good Signadot demo

The validation chain `HTTP POST → driver → Redis → frontend polling` spans three components. No unit test covers it end-to-end. A Signadot sandbox runs the forked driver against baseline frontend/redis — exactly the cross-service, real-cluster validation story from [signadot-integration-spec.md](../../signadot-integration-spec.md).

## Testing

- **Unit (RED→GREEN via opsx apply):** dispatchstore CRUD; arrived handler (404 unknown, 200 idempotent, status transition + notification)
- **Cluster (manual/scripted this pass):** Signadot sandbox with forked driver → `POST arrived` → frontend `/notifications` shows the arrival. This script is the seed of the future `signadot-validate` plan for this behavior.

## Risks

- Driver's HTTP surface is currently health-only; adding a business endpoint may touch k8s Service/port wiring (verify port 8082 exposure in manifests)
- Dispatch record TTL: follow the existing notification pattern (30s SetEx) unless a longer window proves necessary during apply

## Follow-up

- Feature 1 (Driver Status Tracking): second OpenSpec change in the same repo, reusing `dispatchstore`; spec drafted at setup step 4, designed in its own brainstorm before development
- Division-of-work note: step 3 covers the **opsx side** of the integration (task template, N.V step, plan stubs). The Signadot-side runner (`signadot-validate` CLI behavior, plan schema) still follows Joe's answers to the open questions in the integration spec; until then N.V materializes/runs plans manually or via script.
