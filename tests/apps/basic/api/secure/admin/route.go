package admin

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

var Route = DefineRoute(RouteHandlers{
	Middleware: RouteMiddleware{
		All: func(ctx context.Context, r *http.Request, next MiddlewareNext) (any, error) {
			return next(ctx, MiddlewareAllContext{
				IsAdmin:     true,
				Permissions: []string{"read", "write", "delete"},
			})
		},
		Post: func(ctx context.Context, r *http.Request, req PostRequest, mw MiddlewareContext, next PostNext) (PostResponse, error) {
			if !mw.IsAdmin {
				return PostStatus403{Body: "Forbidden: Admin access required"}, nil
			}
			return next(ctx, req)
		},
	},
	Get: func(ctx context.Context, req GetRequest, mw GetContext) (GetResponse, error) {
		return GetStatus200{Body: fmt.Sprintf("%s:%s:%t:%s", mw.UserID, mw.TraceID, mw.IsAdmin, strings.Join(mw.Permissions, ","))}, nil
	},
	Post: func(ctx context.Context, req PostRequest, mw PostContext) (PostResponse, error) {
		return PostStatus201{Body: fmt.Sprintf("%s:%s:%t:%s", req.Body.Data, mw.UserID, mw.IsAdmin, strings.Join(mw.Permissions, ","))}, nil
	},
})
