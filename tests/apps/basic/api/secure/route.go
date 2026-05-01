package secure

import (
	"context"
	"fmt"
)

var Route = DefineRoute(RouteHandlers{
	Middleware: RouteMiddleware{
		All: func(ctx context.Context, next MiddlewareNext) (any, error) {
			return next(ctx, MiddlewareAllContext{
				UserID:  "user-admin",
				TraceID: "trace-root",
			})
		},
	},
	Get: func(ctx context.Context, req GetRequest, mw GetContext) (GetResponse, error) {
		return GetStatus200{Body: fmt.Sprintf("%s:%s", mw.UserID, mw.TraceID)}, nil
	},
})
