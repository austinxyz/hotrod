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
	"testing"
	"time"

	"github.com/signadot/hotrod/pkg/log"
	"github.com/signadot/hotrod/pkg/notifications"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

func TestConsumerStoresDispatchRecord(t *testing.T) {
	store := newTestDispatchStore(t)
	consumer := &Consumer{
		tracer:        noop.NewTracerProvider().Tracer("test"),
		logger:        log.NewFactory(zap.NewNop()),
		dispatchStore: store,
	}
	ctx := context.Background()

	reqCtx := &notifications.RequestContext{ID: 7, SessionID: 42}
	best := &Response{DriverID: "T707365C", ETA: 2 * time.Minute}

	consumer.storeDispatchRecord(ctx, reqCtx, "rk-abc", best)

	got, err := store.Get(ctx, 7)
	if err != nil {
		t.Fatalf("Get after storeDispatchRecord: %v", err)
	}
	if got.SessionID != 42 || got.DriverID != "T707365C" || got.ETA != 2*time.Minute ||
		got.RoutingKey != "rk-abc" || got.Status != StatusDispatched {
		t.Fatalf("record mismatch: %+v", got)
	}
}
