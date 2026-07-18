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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/signadot/hotrod/pkg/log"
	"github.com/signadot/hotrod/pkg/notifications"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

// fakeNotifier records Store calls; NotificationContext mirrors the real
// handler's construction minus env-derived fields.
type fakeNotifier struct {
	stored []notifications.Notification
}

func (f *fakeNotifier) NotificationContext(reqCtx *notifications.RequestContext,
	routingKey string) *notifications.NotificationContext {
	return &notifications.NotificationContext{Request: reqCtx, RoutingKey: routingKey}
}

func (f *fakeNotifier) Store(_ context.Context, n *notifications.Notification) error {
	f.stored = append(f.stored, *n)
	return nil
}

func (f *fakeNotifier) List(_ context.Context, _ uint, _ int) (*notifications.NotificationList, error) {
	return &notifications.NotificationList{}, nil
}

func newArrivedTestServer(t *testing.T) (DispatchStore, *fakeNotifier, *httptest.Server) {
	t.Helper()
	store := newTestDispatchStore(t)
	notifier := &fakeNotifier{}
	handler := NewArrivedHandler(noop.NewTracerProvider(), log.NewFactory(zap.NewNop()), store, notifier)
	mux := http.NewServeMux()
	mux.Handle("POST /dispatches/{requestID}/arrived", handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return store, notifier, srv
}

func TestArrivedTransition(t *testing.T) {
	store, notifier, srv := newArrivedTestServer(t)
	ctx := context.Background()

	rec := &DispatchRecord{
		SessionID:  42,
		DriverID:   "T707365C",
		RoutingKey: "rk-abc",
		Status:     StatusDispatched,
	}
	if err := store.Store(ctx, 7, rec); err != nil {
		t.Fatalf("Store: %v", err)
	}

	resp, err := http.Post(srv.URL+"/dispatches/7/arrived", "application/json", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	got, err := store.Get(ctx, 7)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != StatusArrived {
		t.Fatalf("record status = %q, want %q", got.Status, StatusArrived)
	}

	if len(notifier.stored) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifier.stored))
	}
	n := notifier.stored[0]
	if n.ID != "req-7-arrived" {
		t.Fatalf("notification ID = %q, want %q", n.ID, "req-7-arrived")
	}
	if n.Body != "Driver T707365C arrived at pickup" {
		t.Fatalf("notification body = %q", n.Body)
	}
	if n.Context == nil || n.Context.Request == nil ||
		n.Context.Request.SessionID != 42 || n.Context.Request.ID != 7 ||
		n.Context.RoutingKey != "rk-abc" {
		t.Fatalf("notification context not rebuilt from record: %+v", n.Context)
	}
}

func TestArrivedIdempotent(t *testing.T) {
	store, notifier, srv := newArrivedTestServer(t)
	ctx := context.Background()

	rec := &DispatchRecord{
		SessionID:  42,
		DriverID:   "T707365C",
		RoutingKey: "rk-abc",
		Status:     StatusArrived, // already arrived
	}
	if err := store.Store(ctx, 7, rec); err != nil {
		t.Fatalf("Store: %v", err)
	}

	resp, err := http.Post(srv.URL+"/dispatches/7/arrived", "application/json", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if len(notifier.stored) != 0 {
		t.Fatalf("repeated arrival must not store a new notification, got %d", len(notifier.stored))
	}
}

func TestArrivedUnknownDispatch(t *testing.T) {
	_, notifier, srv := newArrivedTestServer(t)

	resp, err := http.Post(srv.URL+"/dispatches/999/arrived", "application/json", nil)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	if len(notifier.stored) != 0 {
		t.Fatalf("expected no notifications, got %d", len(notifier.stored))
	}
}
