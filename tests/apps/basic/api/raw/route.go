package raw

import (
	"context"
	"io"
	"net/http"
)

var Route = DefineRoute(RouteHandlers{
	Get: func(ctx context.Context, req GetRequest) (GetResponse, error) {
		return RawResponseFunc(func(w http.ResponseWriter, r *http.Request) error {
			w.Header().Set("content-type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, err := io.WriteString(w, "chunk-1\nchunk-2\n")
			return err
		}), nil
	},
})
