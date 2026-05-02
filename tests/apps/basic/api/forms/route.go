package forms

import (
	"context"
	"fmt"
)

var Route = DefineRoute(RouteHandlers{
	Get: func(ctx context.Context, req GetRequest) (GetResponse, error) {
		return GetStatus200{
			Header: TextHeader{ContentType: "text/custom"},
			Body:   "plain",
		}, nil
	},
	Post: func(ctx context.Context, req PostRequest) (PostResponse, error) {
		return PostStatus201{Body: fmt.Sprintf("%s:%s:%d:%t:%d", req.Body.Name, req.Body.Alias, req.Body.Age, req.Body.Active, len(req.Body.Scores))}, nil
	},
	Put: func(ctx context.Context, req PutRequest) (PutResponse, error) {
		return PutStatus200{Body: fmt.Sprintf("%s:%d", req.Body.Title, req.Body.Count)}, nil
	},
	Patch: func(ctx context.Context, req PatchRequest) (PatchResponse, error) {
		return PatchStatus200{Body: []byte{1, 2, 3}}, nil
	},
	Delete: func(ctx context.Context, req DeleteRequest) (DeleteResponse, error) {
		return DeleteStatus200{Body: MultipartResponseBody{Name: "alice", Count: 2}}, nil
	},
})
