package nest

import (
	"context"
	"net/http"
)

var Route = DefineRoute(RouteHandlers{
	Middleware: RouteMiddleware{
		All: func(ctx context.Context, r *http.Request, next MiddlewareNext) (any, error) {
			return next(ctx, MiddlewareAllContext{TraceID: "nested-trace"})
		},
	},
})
