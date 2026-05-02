package secure

import (
	"context"
	"fmt"
	"net/http"
)

var Route = DefineRoute(RouteHandlers{
	Middleware: RouteMiddleware{
		All: func(ctx context.Context, r *http.Request, next MiddlewareNext) (any, error) {
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
