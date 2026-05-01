package slug

import (
	"context"
	"strings"
)

var Route = DefineRoute(RouteHandlers{
	Get: func(ctx context.Context, req GetRequest) (GetResponse, error) {
		return GetStatus200{Body: strings.Join(req.Param, "/")}, nil
	},
})
