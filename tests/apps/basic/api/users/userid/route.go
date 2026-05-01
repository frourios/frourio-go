package userid

import (
	"context"
	"fmt"
)

var Route = DefineRoute(RouteHandlers{
	Get: func(ctx context.Context, req GetRequest) (GetResponse, error) {
		if req.Param == 404 {
			return GetStatus404{Body: "not found"}, nil
		}
		return GetStatus200{Body: fmt.Sprintf("user:%d", req.Param)}, nil
	},
})
