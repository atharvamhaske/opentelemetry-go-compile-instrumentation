// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"net/http"

	"github.com/gorilla/mux"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"

	httpsemconv "go.opentelemetry.io/otelc/instrumentation/net/http/semconv"
	"go.opentelemetry.io/otelc/pkg/hook"
)

// AfterMatch runs after (*Router).Match. The after hook sees only the bool
// return; the request and RouteMatch are read from the hook context params.
//
// Match returns true for a custom NotFoundHandler or MethodNotAllowedHandler
// as well as for a real route. Those cases set MatchErr (ErrNotFound /
// ErrMethodMismatch) and must not receive http.route — same as a 404/405
// that used mux's default handlers.
func AfterMatch(ictx hook.HookContext, _ bool) {
	if !enabler.Enable() {
		return
	}

	req, match := requestAndMatch(ictx)
	if req == nil || match == nil || match.MatchErr != nil || match.Route == nil {
		return
	}

	route, err := match.Route.GetPathTemplate()
	if err != nil || route == "" {
		return
	}

	span := trace.SpanFromContext(req.Context())
	if !span.IsRecording() {
		return
	}

	span.SetName(httpsemconv.HTTPServerSpanName(req.Method, route))
	span.SetAttributes(semconv.HTTPRoute(route))
	logger.Debug("mux route resolved", "route", route)
}

func requestAndMatch(ictx hook.HookContext) (*http.Request, *mux.RouteMatch) {
	if ictx == nil {
		return nil, nil
	}
	var req *http.Request
	var match *mux.RouteMatch
	for i := 0; i < ictx.GetParamCount(); i++ {
		switch p := ictx.GetParam(i).(type) {
		case *http.Request:
			req = p
		case *mux.RouteMatch:
			match = p
		}
	}
	return req, match
}
