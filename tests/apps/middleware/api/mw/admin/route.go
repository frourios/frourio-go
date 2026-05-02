package admin

import (
	"context"
	"net/http"
	"strings"
)

var Route = DefineRoute(RouteHandlers{
	Middleware: RouteMiddleware{
		All: func(ctx context.Context, r *http.Request, next MiddlewareNext) (any, error) {
			isAdmin, permissions := resolveAdmin(r)
			return next(ctx, MiddlewareAllContext{
				IsAdmin:     isAdmin,
				Permissions: permissions,
			})
		},
		Post: func(ctx context.Context, r *http.Request, req PostRequest, mw MiddlewareContext, next PostNext) (PostResponse, error) {
			if !mw.IsAdmin {
				return PostStatus403{Body: ForbiddenBody{Message: "Forbidden: Admin access required"}}, nil
			}
			return next(ctx, req)
		},
	},
	Get: func(ctx context.Context, req GetRequest, mw GetContext) (GetResponse, error) {
		return GetStatus200{Body: AdminContextResponse{
			UserID:      mw.UserID,
			TraceID:     mw.TraceID,
			IsAdmin:     mw.IsAdmin,
			Permissions: mw.Permissions,
		}}, nil
	},
	Post: func(ctx context.Context, req PostRequest, mw PostContext) (PostResponse, error) {
		return PostStatus201{Body: AdminPostResponseBody{
			Received: req.Body.Data,
			Context: AdminContextResponse{
				UserID:      mw.UserID,
				TraceID:     mw.TraceID,
				IsAdmin:     mw.IsAdmin,
				Permissions: mw.Permissions,
			},
		}}, nil
	},
})

func resolveAdmin(r *http.Request) (bool, []string) {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return false, []string{}
	}
	userID := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if userID == "user-admin" {
		return true, []string{"read", "write", "delete"}
	}
	return false, []string{"read"}
}
