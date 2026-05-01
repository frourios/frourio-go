package path

import (
	"context"
	"strings"
)

var Route = DefineRoute(RouteHandlers{
	Get: func(ctx context.Context, req GetRequest) (GetResponse, error) {
		if req.Param == nil {
			return GetStatus200{Body: "root"}, nil
		}
		return GetStatus200{Body: strings.Join(*req.Param, "/")}, nil
	},
})
