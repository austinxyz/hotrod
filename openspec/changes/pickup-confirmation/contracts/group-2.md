# Contract — Group 2: Arrival flow (consumer write + HTTP endpoint + k8s Service)

- **Spec**: "Record written after successful dispatch" (consumer); "The driver service SHALL expose `POST /dispatches/{requestID}/arrived` ... set status to `arrived` and store a notification 'Driver <driverID> arrived at pickup' ... respond 200"; "Unknown dispatch rejected" (404, no notification); "Repeated arrival calls SHALL respond 200 without storing a duplicate notification" (deterministic ID `req-<requestID>-arrived`); "The driver Deployment SHALL expose port 8082 via a k8s Service"; "arrival notification SHALL flow through the existing notification store so the frontend's /notifications polling surfaces it with zero frontend changes"
- **Runtime**: validated by signadot plan `pickup-confirmation-arrival`
- **Code**: handler registered on existing :8082 mux in processor.go (no second server); NotificationContext rebuilt from record fields only (no baggage on HTTP POST); GET+SET race accepted — deterministic notification ID dedupes; k8s Service added in k8s/base/driver.yaml so all overlays inherit
- **Threshold**: 80
