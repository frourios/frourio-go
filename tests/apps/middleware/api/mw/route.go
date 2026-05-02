package mw

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

var Route = DefineRoute(RouteHandlers{
	Middleware: RouteMiddleware{
		All: func(ctx context.Context, r *http.Request, next MiddlewareNext) (any, error) {
			ctxValue := MiddlewareAllContext{TraceID: traceIDFromHeader(r)}
			if userID := bearerUserID(r); userID != "" {
				ctxValue.UserID = &userID
			}
			return next(ctx, ctxValue)
		},
	},
	Get: func(ctx context.Context, req GetRequest, mw GetContext) (GetResponse, error) {
		return GetStatus200{Body: GetResponseBody{UserID: mw.UserID, TraceID: mw.TraceID}}, nil
	},
})

func bearerUserID(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func traceIDFromHeader(r *http.Request) string {
	if id := r.Header.Get("X-Trace-Id"); id != "" {
		return id
	}
	return generateTraceID()
}

func generateTraceID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "fallback"
	}
	return hex.EncodeToString(buf[:])
}
