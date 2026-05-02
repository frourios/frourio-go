# Testing

Tests run against the generated `api.Handler()` in two flavors:

1. **In-process** — `httptest.NewRecorder` + `Handler().ServeHTTP(w, r)`.
   Fastest, no network. Use for most unit tests.
2. **Loopback HTTP** — `httptest.NewServer(api.Handler())`. Required when
   testing against an HTTP-aware client (e.g., the oapi-codegen client).

## In-process Test

```go
func TestUsersGet(t *testing.T) {
    req := httptest.NewRequestWithContext(
        context.Background(),
        http.MethodGet,
        "/api/users?limit=1",
        nil,
    )
    res := httptest.NewRecorder()
    api.Handler().ServeHTTP(res, req)

    if res.Code != http.StatusOK {
        t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
    }
    if res.Body.String() != `{"body":["alice"]}` {
        t.Fatalf("body = %q", res.Body.String())
    }
}
```

`httptest.NewRequestWithContext` propagates a real context — preferred over
`httptest.NewRequest` so middleware that reads context values behaves.

## Sending Bodies

### JSON

```go
body := strings.NewReader(`{"name":"alice","age":20}`)
req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/users", body)
req.Header.Set("content-type", "application/json")
```

### URL-encoded

```go
body := strings.NewReader("name=alice&age=20&active=true&score=1.5&score=2.5")
req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/forms", body)
req.Header.Set("content-type", "application/x-www-form-urlencoded")
```

### Multipart form-data

```go
var buf bytes.Buffer
w := multipart.NewWriter(&buf)
_ = w.WriteField("title", "report")
_ = w.WriteField("count", "3")
_ = w.Close()

req := httptest.NewRequestWithContext(ctx, http.MethodPut, "/api/forms", &buf)
req.Header.Set("content-type", w.FormDataContentType())
```

## Asserting Responses

### Status + plain text

```go
if res.Code != http.StatusOK {
    t.Fatalf("status = %d", res.Code)
}
if got := res.Body.String(); got != "expected" {
    t.Fatalf("body = %q", got)
}
```

### JSON body

```go
var body struct {
    UserID  *string `json:"userId"`
    TraceID string  `json:"traceId"`
}
if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
    t.Fatalf("decode: %v", err)
}
```

### Custom Content-Type / Header

```go
if got := res.Header().Get("content-type"); got != "text/custom" {
    t.Fatalf("content-type = %q", got)
}
```

### Binary body

```go
if !bytes.Equal(res.Body.Bytes(), []byte{1, 2, 3}) {
    t.Fatalf("body = %v", res.Body.Bytes())
}
```

### Multipart response

```go
ct := res.Header().Get("content-type")
if !strings.HasPrefix(ct, "multipart/form-data; boundary=") {
    t.Fatalf("content-type = %q", ct)
}
if !strings.Contains(res.Body.String(), `name="name"`) {
    t.Fatalf("missing form field")
}
```

## Testing Validation Errors

Send malformed input and assert 422 plus the structured error envelope:

```go
req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/users/abc", nil)
res := httptest.NewRecorder()
api.Handler().ServeHTTP(res, req)

if res.Code != http.StatusUnprocessableEntity {
    t.Fatalf("status = %d", res.Code)
}

var body struct {
    Status int    `json:"status"`
    Error  string `json:"error"`
    Issues []struct {
        Path    []any  `json:"path"`     // ["param"] or ["query", "limit"], etc.
        Message string `json:"message"`  // validator tag name
    } `json:"issues"`
}
_ = json.Unmarshal(res.Body.Bytes(), &body)
if body.Status != 422 {
    t.Fatalf("envelope.status = %d", body.Status)
}
```

## Testing Middleware

### Authenticated request

```go
req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/mw/admin", nil)
req.Header.Set("Authorization", "Bearer user-admin")
api.Handler().ServeHTTP(res, req)
```

### Unauthenticated → 403

```go
req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/mw/admin",
    strings.NewReader(`{"data":"hi"}`))
req.Header.Set("content-type", "application/json")
api.Handler().ServeHTTP(res, req)

if res.Code != http.StatusForbidden {
    t.Fatalf("status = %d", res.Code)
}
```

### Inherited context

Middleware-set context fields appear in the response when the handler echoes
them. For deep routes like `/api/secure/admin/users?role=admin`, decode the
JSON and assert `UserID`, `TraceID`, `IsAdmin`, `Permissions`, etc.

## Table-Driven Tests

frourio-go is well-suited to table-driven tests because most error paths
follow the same shape (request → status code, optionally response body
substring).

```go
cases := []struct {
    name       string
    body       string
    wantStatus int
}{
    {"missing name", `{}`, 422},
    {"name not string", `{"name":1}`, 422},
    {"empty name", `{"name":""}`, 422},
    {"valid", `{"name":"alice"}`, 201},
}

for _, tc := range cases {
    t.Run(tc.name, func(t *testing.T) {
        req := httptest.NewRequestWithContext(ctx, http.MethodPost,
            "/api/users", strings.NewReader(tc.body))
        req.Header.Set("content-type", "application/json")
        res := httptest.NewRecorder()
        api.Handler().ServeHTTP(res, req)
        if res.Code != tc.wantStatus {
            t.Fatalf("status = %d, want %d", res.Code, tc.wantStatus)
        }
    })
}
```

> Lint note: `golangci-lint`'s `fieldalignment` will complain about poor
> field ordering in inline struct types. Put pointer / wider fields before
> narrower ones — e.g., `name string` last after fixed-size fields, or
> reorder slice/struct fields before `bool`.

## Testing With the OpenAPI Client

`httptest.NewServer` returns a real loopback URL the client can hit:

```go
func TestOpenAPIClient(t *testing.T) {
    server := httptest.NewServer(api.Handler())
    defer server.Close()

    client, err := openapiclient.NewClientWithResponses(
        server.URL,
        openapiclient.WithHTTPClient(server.Client()),
    )
    if err != nil {
        t.Fatal(err)
    }

    limit := 1
    res, err := client.GetApiUsersWithResponse(
        context.Background(),
        &openapiclient.GetApiUsersParams{Limit: &limit},
    )
    if err != nil {
        t.Fatal(err)
    }
    if res.StatusCode() != http.StatusOK {
        t.Fatalf("status = %d", res.StatusCode())
    }
}
```

Each operation in the generated client has both a typed-response method
(`...WithResponse`) and a raw method. The typed method exposes both the
success body and any documented error bodies (such as `JSON422` for
`FrourioError`).

## Real Examples in This Repo

- Comprehensive feature tests:
  [tests/apps/basic/api/route_test.go](../../tests/apps/basic/api/route_test.go)
- OpenAPI client coverage:
  [tests/apps/basic/client_test.go](../../tests/apps/basic/client_test.go)
- Middleware and auth scenarios:
  [tests/apps/middleware/api/route_test.go](../../tests/apps/middleware/api/route_test.go)

## Cross-Reference

- Decode + validation behavior under test: [validation.md](validation.md)
- Status codes and bodies: [response.md](response.md)
- Middleware composition under test: [middleware.md](middleware.md)
