package users

import (
	"context"

	"github.com/frourios/frourio-go/tests/apps/middleware/api/mw/admin"
)

var Route = DefineRoute(RouteHandlers{
	Get: func(ctx context.Context, req GetRequest, mw GetContext) (GetResponse, error) {
		users := []string{}
		if req.Query.Role != nil && *req.Query.Role == "admin" && mw.IsAdmin {
			users = []string{"admin1"}
		}
		return GetStatus200{Body: UsersResponseBody{
			Context: admin.AdminContextResponse{
				UserID:      mw.UserID,
				TraceID:     mw.TraceID,
				IsAdmin:     mw.IsAdmin,
				Permissions: mw.Permissions,
			},
			Users: users,
		}}, nil
	},
})
