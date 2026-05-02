package postid

import (
	"context"
	"fmt"
)

var Route = DefineRoute(RouteHandlers{
	Get: func(ctx context.Context, req GetRequest) (GetResponse, error) {
		return GetStatus200{Body: fmt.Sprintf("user:%d/post:%s", req.Params.Userid, req.Params.Postid)}, nil
	},
})
