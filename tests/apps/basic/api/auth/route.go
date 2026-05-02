package auth

import (
	"context"
	"net/http"
)

var Route = DefineRoute(RouteHandlers{
	Middleware: RouteMiddleware{
		All: func(ctx context.Context, r *http.Request, next MiddlewareNext) (any, error) {
			return next(ctx, MiddlewareAllContext{TraceID: "trace-123"})
		},
	},
	Get: func(ctx context.Context, req GetRequest, mw GetContext) (GetResponse, error) {
		return GetStatus200{Body: mw.TraceID}, nil
	},
})
