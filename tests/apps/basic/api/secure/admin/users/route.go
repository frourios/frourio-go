package users

import (
	"context"
	"fmt"
	"strings"
)

var Route = DefineRoute(RouteHandlers{
	Get: func(ctx context.Context, req GetRequest, mw GetContext) (GetResponse, error) {
		role := "all"
		if req.Query.Role != nil {
			role = *req.Query.Role
		}
		return GetStatus200{Body: fmt.Sprintf("%s:%s:%t:%s:%s", mw.UserID, mw.TraceID, mw.IsAdmin, strings.Join(mw.Permissions, ","), role)}, nil
	},
})
