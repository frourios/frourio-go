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
	root, err := client.GetApiWithResponse(ctx, &openapiclient.GetApiParams{
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
		t.Fatalf("GET /api = %d %q", root.StatusCode(), root.Body)
	}

	usersLimit := 1
	users, err := client.GetApiUsersWithResponse(ctx, &openapiclient.GetApiUsersParams{Limit: &usersLimit})
	if err != nil {
		t.Fatal(err)
	}
	if users.StatusCode() != http.StatusOK || users.JSON200 == nil || len(*users.JSON200) != 1 || (*users.JSON200)[0] != "alice" {
		t.Fatalf("GET /api/users = %d %#v body=%q", users.StatusCode(), users.JSON200, users.Body)
	}

	age := 20
	created, err := client.PostApiUsersWithResponse(ctx, openapiclient.PostApiUsersJSONRequestBody{
		Name: "alice",
		Age:  &age,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.StatusCode() != http.StatusCreated || string(created.Body) != "alice:20" {
		t.Fatalf("POST /api/users = %d %q", created.StatusCode(), created.Body)
	}

	user, err := client.GetApiUsersByUseridWithResponse(ctx, 123)
	if err != nil {
		t.Fatal(err)
	}
	if user.StatusCode() != http.StatusOK || string(user.Body) != "user:123" {
		t.Fatalf("GET /api/users/123 = %d %q", user.StatusCode(), user.Body)
	}

	sale, err := client.GetApiProductsWithResponse(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if sale.StatusCode() != http.StatusOK || string(sale.Body) != "sale" {
		t.Fatalf("GET /api/products/セール品 = %d %q", sale.StatusCode(), sale.Body)
	}
}
