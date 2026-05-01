package nest

import "context"

var Route = DefineRoute(RouteHandlers{
	Middleware: RouteMiddleware{
		All: func(ctx context.Context, next MiddlewareNext) (any, error) {
			return next(ctx, MiddlewareAllContext{TraceID: "nested-trace"})
		},
	},
})
