package auth

import "context"

var Route = DefineRoute(RouteHandlers{
	Middleware: RouteMiddleware{
		All: func(ctx context.Context, next MiddlewareNext) (any, error) {
			return next(ctx, MiddlewareAllContext{TraceID: "trace-123"})
		},
	},
	Get: func(ctx context.Context, req GetRequest, mw GetContext) (GetResponse, error) {
		return GetStatus200{Body: mw.TraceID}, nil
	},
})
