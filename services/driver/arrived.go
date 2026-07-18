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
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/signadot/hotrod/pkg/log"
	"github.com/signadot/hotrod/pkg/notifications"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// ArrivedHandler serves POST /dispatches/{requestID}/arrived: it transitions a
// dispatch record to "arrived" and stores the pickup notification.
type ArrivedHandler struct {
	tracer   trace.Tracer
	logger   log.Factory
	store    DispatchStore
	notifier notifications.Interface
}

func NewArrivedHandler(tracerProvider trace.TracerProvider, logger log.Factory,
	store DispatchStore, notifier notifications.Interface) *ArrivedHandler {
	return &ArrivedHandler{
		tracer:   tracerProvider.Tracer("arrived-handler"),
		logger:   logger.With(zap.String("component", "arrived-handler")),
		store:    store,
		notifier: notifier,
	}
}

func (h *ArrivedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "DispatchArrived", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	requestID, err := strconv.ParseUint(r.PathValue("requestID"), 10, 64)
	if err != nil {
		http.Error(w, "invalid request id", http.StatusBadRequest)
		return
	}

	record, err := h.store.Get(ctx, uint(requestID))
	if errors.Is(err, ErrDispatchNotFound) {
		http.Error(w, "dispatch not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.For(ctx).Error("failed to load dispatch record", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if record.Status == StatusArrived {
		// Idempotent repeat: already arrived, nothing to transition or notify.
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.store.UpdateStatus(ctx, uint(requestID), StatusArrived); err != nil {
		h.logger.For(ctx).Error("failed to update dispatch status", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Rebuild the notification context purely from the persisted record — an
	// external HTTP POST carries no Kafka baggage to derive it from.
	notificationCtx := h.notifier.NotificationContext(&notifications.RequestContext{
		ID:        uint(requestID),
		SessionID: record.SessionID,
	}, record.RoutingKey)
	if err := h.notifier.Store(ctx, &notifications.Notification{
		ID:        fmt.Sprintf("req-%d-arrived", requestID),
		Timestamp: time.Now(),
		Context:   notificationCtx,
		Body:      fmt.Sprintf("Driver %s arrived at pickup", record.DriverID),
	}); err != nil {
		h.logger.For(ctx).Error("failed to store arrival notification", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
