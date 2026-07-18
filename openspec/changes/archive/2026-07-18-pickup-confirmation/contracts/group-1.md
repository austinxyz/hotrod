# Contract — Group 1: Dispatch store (Redis-backed record CRUD)

- **Spec**: "When the driver service completes dispatch processing (best-ETA driver selected), it SHALL persist a dispatch record in Redis under key `dispatch:<requestID>` containing sessionID, driverID, eta, routingKey, and status `dispatched`." (store layer: schema + CRUD; the consumer write itself is group 2)
- **Runtime**: `go test ./services/driver/...` → expected: all dispatchstore tests pass (round-trip, missing-key, status update), no build errors
- **Code**: separate `dispatchstore.go` component with small interface (Store/Get/UpdateStatus), own Redis client via pkg/config, mirrors pkg/notifications handler pattern; record includes routingKey + sessionID (NotificationContext reconstruction); TTL 5 min SetEx constant; tests use github.com/alicebob/miniredis/v2 (new test-only dep)
- **Threshold**: 80
