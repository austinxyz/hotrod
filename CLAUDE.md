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

(populated by /opsx:archive)
