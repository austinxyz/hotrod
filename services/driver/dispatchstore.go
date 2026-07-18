// Copyright (c) 2026 Signadot Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/signadot/hotrod/pkg/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	// StatusDispatched is the initial dispatch record state, written when the
	// driver service selects a best-ETA driver.
	StatusDispatched = "dispatched"
	// StatusArrived is set when the driver confirms arrival at pickup.
	StatusArrived = "arrived"

	// dispatchTTL must outlive the demo interaction window so an external
	// arrival POST has a record to hit (unlike the 30s notification TTL).
	dispatchTTL = 5 * time.Minute
)

// ErrDispatchNotFound is returned when no dispatch record exists for a request ID.
var ErrDispatchNotFound = errors.New("dispatch record not found")

// DispatchRecord is the persisted state of one dispatch, keyed by request ID.
// SessionID and RoutingKey are stored so the arrival handler can rebuild a
// notifications.NotificationContext without any Kafka/baggage context.
type DispatchRecord struct {
	SessionID  uint          `json:"sessionID"`
	DriverID   string        `json:"driverID"`
	ETA        time.Duration `json:"eta"`
	RoutingKey string        `json:"routingKey"`
	Status     string        `json:"status"`
}

// DispatchStore persists dispatch records in Redis.
type DispatchStore interface {
	Store(ctx context.Context, requestID uint, record *DispatchRecord) error
	Get(ctx context.Context, requestID uint) (*DispatchRecord, error)
	UpdateStatus(ctx context.Context, requestID uint, status string) error
}

type dispatchStore struct {
	tracer trace.Tracer
	logger log.Factory
	rdb    *redis.Client
}

// NewDispatchStore returns a Redis-backed DispatchStore.
func NewDispatchStore(tracerProvider trace.TracerProvider, logger log.Factory, rdb *redis.Client) DispatchStore {
	return &dispatchStore{
		tracer: tracerProvider.Tracer("dispatchstore"),
		logger: logger.With(zap.String("component", "dispatchstore")),
		rdb:    rdb,
	}
}

func (s *dispatchStore) Store(ctx context.Context, requestID uint, record *DispatchRecord) error {
	ctx, span := s.tracer.Start(ctx, "StoreDispatch", trace.WithSpanKind(trace.SpanKindClient))
	span.SetAttributes(attribute.Key("request.id").Int(int(requestID)))
	defer span.End()

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal dispatch record: %w", err)
	}
	if err := s.rdb.SetEx(ctx, dispatchKey(requestID), data, dispatchTTL).Err(); err != nil {
		return fmt.Errorf("failed to store dispatch record: %w", err)
	}
	s.logger.For(ctx).Info("Stored dispatch record",
		zap.Uint("requestID", requestID), zap.Any("record", *record))
	return nil
}

func (s *dispatchStore) Get(ctx context.Context, requestID uint) (*DispatchRecord, error) {
	ctx, span := s.tracer.Start(ctx, "GetDispatch", trace.WithSpanKind(trace.SpanKindClient))
	span.SetAttributes(attribute.Key("request.id").Int(int(requestID)))
	defer span.End()

	r := s.rdb.Get(ctx, dispatchKey(requestID))
	if r.Err() == redis.Nil {
		return nil, ErrDispatchNotFound
	}
	if r.Err() != nil {
		return nil, fmt.Errorf("failed to get dispatch record: %w", r.Err())
	}
	var record DispatchRecord
	if err := json.Unmarshal([]byte(r.Val()), &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dispatch record: %w", err)
	}
	return &record, nil
}

func (s *dispatchStore) UpdateStatus(ctx context.Context, requestID uint, status string) error {
	ctx, span := s.tracer.Start(ctx, "UpdateDispatchStatus", trace.WithSpanKind(trace.SpanKindClient))
	span.SetAttributes(
		attribute.Key("request.id").Int(int(requestID)),
		attribute.Key("dispatch.status").String(status),
	)
	defer span.End()

	record, err := s.Get(ctx, requestID)
	if err != nil {
		return err
	}
	record.Status = status
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal dispatch record: %w", err)
	}
	// KeepTTL preserves the remaining expiry rather than restarting the window.
	if err := s.rdb.Set(ctx, dispatchKey(requestID), data, redis.KeepTTL).Err(); err != nil {
		return fmt.Errorf("failed to update dispatch record: %w", err)
	}
	s.logger.For(ctx).Info("Updated dispatch status",
		zap.Uint("requestID", requestID), zap.String("status", status))
	return nil
}

func dispatchKey(requestID uint) string {
	return fmt.Sprintf("dispatch:%d", requestID)
}
