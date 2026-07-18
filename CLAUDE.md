# hotrod-opsx

Fork of signadot/hotrod used as the reference example for the OpenSpec + Superpowers × Signadot
integration (collaboration with Signadot — Ani/Joe). Remote `origin` = austinxyz/hotrod,
`upstream` = signadot/hotrod.

## Workflow

All feature work goes through `/opsx:explore → /opsx:propose → /opsx:apply → /opsx:archive`.
Signadot plan/validate wiring is enabled (`openspec/config.yaml` → integrations.signadot).
The opsx skills in `.claude/commands/opsx/` and the schema in `openspec/schemas/superpowers-driven/`
are LOCAL copies extended for Signadot — iterate here; sync back to the opsx-superpowers repo
only after the integration is validated end-to-end.
Integration spec: https://github.com/austinxyz/opsx-superpowers/blob/signadot/docs/signadot-integration-spec.md

## Environment

- kind cluster `hotrod` (kubectl context `kind-hotrod`), baseline deployed in ns `hotrod`
  from `k8s/overlays/prod/devmesh`
- Signadot cluster name: `austin-staging-1` (Operator installed in ns `signadot`)
- Frontend UI: `kubectl -n hotrod port-forward svc/frontend 8080:8080` → http://localhost:8080
- Tests: `go test ./...`

## Planned changes

1. `pickup-confirmation` (Feature 2) — spec: docs/superpowers/specs/2026-07-18-pickup-confirmation-design.md (APPROVED)
2. `driver-status` (Feature 1) — spec: docs/superpowers/specs/2026-07-18-driver-status-design.md (DRAFT — brainstorm before development)

## Pitfalls

- `dispatch:<requestID>` keys collide across browser sessions — the frontend's request
  counter restarts at 1 per session, so a new session's dispatch overwrites the record.
  Fine at demo scale; a real system (and the driver-status change) needs a globally
  unique dispatch id.
- Signadot plans have no sleep primitive and `request-http` exits 1 on transport
  timeout (blackhole-URL delays fail the step). For async waits, chain request-http
  steps via `extraInputs` refs and space them with a slow-but-answering endpoint
  (e.g. httpbin.org/delay/4).
- Plan execution requires Managed Plan Runners enabled per cluster in the dashboard
  (Platform → Managed Runners); a `jobrunnergroup` does NOT satisfy "no plan runner
  group on cluster". The org image allowlist rejected all run-container/k6 images even
  after allowlisting (reported to Signadot) — only actionbox-backed actions
  (request-http/eval/check) run without it.
- Frontend toast notifications expire after 30s (Redis SetEx) — for demos, the UI's
  Logs panel retains the full event chain; don't debug "missing" notifications
  against the toast area.

## Kickoff (next steps)

1. `/opsx:explore pickup-confirmation` — requirements distill from
   docs/superpowers/specs/2026-07-18-pickup-confirmation-design.md (design already approved;
   explore should be fast: draft requirements → review → REVIEWED)
2. `/opsx:propose pickup-confirmation` — expect step 3b to author
   signadot-plans/<behavior-id>.yaml with unbound params
3. `/opsx:apply pickup-confirmation` — N.V VALIDATE binds params and runs against
   austin-staging-1 (manual/scripted until the signadot-validate CLI contract is confirmed with Joe)
4. `/opsx:archive pickup-confirmation` — registers the plan into openspec/specs/*/plans/
5. After e2e validated: sync skill changes back to opsx-superpowers (see plan's Sync-back section)
