package users

import (
	"context"
	"fmt"
)

var Route = DefineRoute(RouteHandlers{
	Get: func(ctx context.Context, req GetRequest) (GetResponse, error) {
		if req.Query.Limit != nil && *req.Query.Limit == 1 {
			return GetStatus200{Body: []string{"alice"}}, nil
		}
		return GetStatus200{Body: []string{"alice", "bob"}}, nil
	},
	Post: func(ctx context.Context, req PostRequest) (PostResponse, error) {
		if req.Body.Name == "bad" {
			return PostStatus400{Body: "bad user"}, nil
		}
		if req.Body.Age != nil {
			return PostStatus201{Body: fmt.Sprintf("%s:%d", req.Body.Name, *req.Body.Age)}, nil
		}
		return PostStatus201{Body: req.Body.Name}, nil
	},
})
