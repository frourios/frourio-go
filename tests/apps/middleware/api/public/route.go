package public

import "context"

var Route = DefineRoute(RouteHandlers{
	Get: func(ctx context.Context, req GetRequest) (GetResponse, error) {
		return GetStatus200{Body: PublicResponseBody{Message: "This is a public endpoint."}}, nil
	},
})
