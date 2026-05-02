package path

import (
	"context"
	"strings"
)

var Route = DefineRoute(RouteHandlers{
	Get: func(ctx context.Context, req GetRequest) (GetResponse, error) {
		if req.Params.Path == nil {
			return GetStatus200{Body: "root"}, nil
		}
		return GetStatus200{Body: strings.Join(*req.Params.Path, "/")}, nil
	},
})
