## ADDED Requirements

### Requirement: Dispatch record persisted at dispatch time
When the driver service completes dispatch processing (best-ETA driver selected), it SHALL persist a dispatch record in Redis under key `dispatch:<requestID>` containing sessionID, driverID, eta, routingKey, and status `dispatched`.

#### Scenario: Record written after successful dispatch
- **WHEN** `processDispatchRequest` selects a best driver for request ID R
- **THEN** Redis contains `dispatch:R` with the selected driverID, the request's sessionID and routingKey, the computed ETA, and status `dispatched`

#### Scenario: Dispatch failure writes no record
- **WHEN** dispatch processing fails before a driver is selected (e.g. best-ETA error)
- **THEN** no `dispatch:<requestID>` key is written

### Requirement: Arrival endpoint transitions dispatch state
The driver service SHALL expose `POST /dispatches/{requestID}/arrived` on its existing HTTP server (:8082). For a record in status `dispatched`, the endpoint SHALL set status to `arrived` and store a notification "Driver <driverID> arrived at pickup" for the record's session, then respond 200.

#### Scenario: Successful arrival transition
- **WHEN** `POST /dispatches/R/arrived` is called and `dispatch:R` has status `dispatched`
- **THEN** `dispatch:R` status becomes `arrived` and a notification "Driver <driverID> arrived at pickup" is stored for the record's sessionID, response 200

#### Scenario: Unknown dispatch rejected
- **WHEN** `POST /dispatches/R/arrived` is called and `dispatch:R` does not exist
- **THEN** the endpoint responds 404 and stores no notification

### Requirement: Arrival is idempotent
Repeated arrival calls for the same dispatch SHALL respond 200 without storing a duplicate notification. The arrival notification SHALL use the deterministic ID `req-<requestID>-arrived` so the notification handler's existing ID-dedupe applies.

#### Scenario: Repeated arrival call
- **WHEN** `POST /dispatches/R/arrived` is called and `dispatch:R` already has status `arrived`
- **THEN** the endpoint responds 200 and the session's notification list still contains exactly one `req-R-arrived` notification

### Requirement: Arrival visible via existing frontend polling
The arrival notification SHALL flow through the existing notification store so the frontend's `/notifications` polling surfaces it with zero frontend changes.

#### Scenario: End-to-end arrival visibility
- **WHEN** a ride is requested through the frontend, and `POST /dispatches/<requestID>/arrived` is subsequently called on the driver service
- **THEN** a following frontend `GET /notifications` for that session includes "Driver <driverID> arrived at pickup"

### Requirement: Driver endpoint reachable in-cluster
The driver Deployment SHALL expose port 8082 via a k8s Service so the arrival endpoint is callable inside the cluster (and by Signadot sandbox validation).

#### Scenario: In-cluster call succeeds
- **WHEN** a pod in the cluster sends `POST http://driver:8082/dispatches/R/arrived`
- **THEN** the request reaches the driver service and is handled per the arrival requirements
