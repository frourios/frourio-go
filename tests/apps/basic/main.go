package main

//go:generate go run ../../.. generate ./api --openapi ./openapi.json

import (
	"log"
	"net/http"
	"time"

	"github.com/frourios/frourio-go/tests/apps/basic/api"
)

func main() {
	server := &http.Server{
		Addr:              ":8080",
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
