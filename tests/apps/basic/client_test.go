package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/frourios/frourio-go/tests/apps/basic/api"
	"github.com/frourios/frourio-go/tests/apps/basic/openapiclient"
)

func TestOpenAPIClientRequestsServer(t *testing.T) {
	server := httptest.NewServer(api.Handler())
	defer server.Close()

	client, err := openapiclient.NewClientWithResponses(server.URL, openapiclient.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	search := "hello"
	limit := 10
	rawName := "raw"
	active := true
	root, err := client.GetWithResponse(ctx, &openapiclient.GetParams{
		Search:  &search,
		Limit:   &limit,
		RawName: &rawName,
		Active:  &active,
		Score:   []float32{1.5, 2.5},
	})
	if err != nil {
		t.Fatal(err)
	}
	if root.StatusCode() != http.StatusOK || string(root.Body) != "ok" {
		t.Fatalf("GET / = %d %q", root.StatusCode(), root.Body)
	}

	usersLimit := 1
	users, err := client.GetUsersWithResponse(ctx, &openapiclient.GetUsersParams{Limit: &usersLimit})
	if err != nil {
		t.Fatal(err)
	}
	if users.StatusCode() != http.StatusOK || users.JSON200 == nil || len(*users.JSON200) != 1 || (*users.JSON200)[0] != "alice" {
		t.Fatalf("GET /users = %d %#v body=%q", users.StatusCode(), users.JSON200, users.Body)
	}

	age := 20
	created, err := client.PostUsersWithResponse(ctx, openapiclient.PostUsersJSONRequestBody{
		Name: "alice",
		Age:  &age,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.StatusCode() != http.StatusCreated || string(created.Body) != "alice:20" {
		t.Fatalf("POST /users = %d %q", created.StatusCode(), created.Body)
	}

	user, err := client.GetUsersByUseridWithResponse(ctx, 123)
	if err != nil {
		t.Fatal(err)
	}
	if user.StatusCode() != http.StatusOK || string(user.Body) != "user:123" {
		t.Fatalf("GET /users/123 = %d %q", user.StatusCode(), user.Body)
	}

	sale, err := client.GetProductsWithResponse(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if sale.StatusCode() != http.StatusOK || string(sale.Body) != "sale" {
		t.Fatalf("GET /products/セール品 = %d %q", sale.StatusCode(), sale.Body)
	}
}
