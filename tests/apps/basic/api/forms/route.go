package forms

import (
	"context"
	"fmt"
)

var Route = DefineRoute(RouteHandlers{
	Post: func(ctx context.Context, req PostRequest) (PostResponse, error) {
		return PostStatus201{Body: fmt.Sprintf("%s:%d:%t:%d", req.Body.Name, req.Body.Age, req.Body.Active, len(req.Body.Scores))}, nil
	},
	Put: func(ctx context.Context, req PutRequest) (PutResponse, error) {
		return PutStatus200{Body: fmt.Sprintf("%s:%d", req.Body.Title, req.Body.Count)}, nil
	},
})
