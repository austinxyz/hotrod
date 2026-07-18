# Design: HotROD Driver Status Tracking (DRAFT)

**Status:** DRAFT — needs its own brainstorm pass before development. Do not /opsx:propose from this.
**Date:** 2026-07-18
**Depends on:** pickup-confirmation (reuses services/driver dispatchstore)

## Sketch (from service-map analysis, Option 1)

User-visible: frontend shows driver availability — "Driver busy" / "Driver accepting rides".

- Span: frontend ↔ driver ↔ location
- Driver publishes status transitions (dispatched → busy → available) building on the
  dispatch records introduced by pickup-confirmation
- Frontend surfaces status via the existing notification/polling mechanism or a new
  lightweight status endpoint (DECIDE at brainstorm)

## Open questions for the brainstorm

1. Status storage: extend `dispatch:<id>` records vs a separate `driver:<id>:status` key
2. Who transitions status back to available — timer, explicit API, or ride-complete event?
3. Does location service need on-duty/off-duty awareness, or is that scope creep? (YAGNI check)
4. Signadot plan: which cross-service assertion proves the behavior end-to-end?
