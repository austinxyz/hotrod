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
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/signadot/hotrod/pkg/log"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

func newTestDispatchStore(t *testing.T) DispatchStore {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return NewDispatchStore(noop.NewTracerProvider(), log.NewFactory(zap.NewNop()), rdb)
}

func TestDispatchStoreRoundTrip(t *testing.T) {
	store := newTestDispatchStore(t)
	ctx := context.Background()

	rec := &DispatchRecord{
		SessionID:  42,
		DriverID:   "T707365C",
		ETA:        2 * time.Minute,
		RoutingKey: "rk-abc",
		Status:     StatusDispatched,
	}
	if err := store.Store(ctx, 7, rec); err != nil {
		t.Fatalf("Store: %v", err)
	}

	got, err := store.Get(ctx, 7)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.SessionID != 42 || got.DriverID != "T707365C" || got.ETA != 2*time.Minute ||
		got.RoutingKey != "rk-abc" || got.Status != StatusDispatched {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestDispatchStoreGetUnknown(t *testing.T) {
	store := newTestDispatchStore(t)

	_, err := store.Get(context.Background(), 999)
	if !errors.Is(err, ErrDispatchNotFound) {
		t.Fatalf("expected ErrDispatchNotFound, got %v", err)
	}
}

func TestDispatchStoreUpdateStatus(t *testing.T) {
	store := newTestDispatchStore(t)
	ctx := context.Background()

	rec := &DispatchRecord{
		SessionID:  42,
		DriverID:   "T707365C",
		ETA:        2 * time.Minute,
		RoutingKey: "rk-abc",
		Status:     StatusDispatched,
	}
	if err := store.Store(ctx, 7, rec); err != nil {
		t.Fatalf("Store: %v", err)
	}

	if err := store.UpdateStatus(ctx, 7, StatusArrived); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got, err := store.Get(ctx, 7)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Status != StatusArrived {
		t.Fatalf("status = %q, want %q", got.Status, StatusArrived)
	}
	if got.DriverID != "T707365C" || got.SessionID != 42 {
		t.Fatalf("other fields lost on update: %+v", got)
	}
}

func TestDispatchStoreUpdateStatusUnknown(t *testing.T) {
	store := newTestDispatchStore(t)

	err := store.UpdateStatus(context.Background(), 999, StatusArrived)
	if !errors.Is(err, ErrDispatchNotFound) {
		t.Fatalf("expected ErrDispatchNotFound, got %v", err)
	}
}
