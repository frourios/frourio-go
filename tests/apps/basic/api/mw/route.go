package mw

import (
	"context"
	"net/http"
)

var Route = DefineRoute(RouteHandlers{
	Middleware: RouteMiddleware{
		All: func(ctx context.Context, r *http.Request, next MiddlewareNext) (any, error) {
			return next(context.WithValue(ctx, roleKey{}, "admin"))
		},
		Get: func(ctx context.Context, r *http.Request, req GetRequest, mw MiddlewareContext, next GetNext) (GetResponse, error) {
			role, _ := ctx.Value(roleKey{}).(string)
			if role == "" {
				return GetStatus403{Body: "forbidden"}, nil
			}
			return next(ctx, req, GetMiddlewareContext{Role: role})
		},
	},
	Get: func(ctx context.Context, req GetRequest, mw GetContext) (GetResponse, error) {
		return GetStatus200{Body: mw.Role}, nil
	},
})

type roleKey struct{}
