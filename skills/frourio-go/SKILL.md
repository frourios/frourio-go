---
name: frourio-go
description: Use this skill when adding, editing, testing, or explaining frourio-go APIs. It covers frourio.go specs, route.go handlers, generated relay/server files, OpenAPI output, validation, path params, middleware, response formats, raw responses, and client-generation checks.
---

# frourio-go

frourio-go builds typed Go HTTP APIs from route-local `frourio.go` specs and
hand-written `route.go` handlers, plus generated glue files and OpenAPI output.

Use this skill when the user asks to:

- add or change an endpoint
- define request or response types (Param, Query, Header, Body)
- add path parameters, catch-all, optional catch-all, or `FrourioPath` URL overrides
- add validation, comments-as-OpenAPI metadata
- add middleware (route-wide `All`, method-specific, ancestor inheritance)
- expose URL-encoded forms, multipart form-data, raw responses, or custom content types
- regenerate `frourio_relay.go`, `frourio_server.go`, or `openapi.json`
- generate OpenAPI clients via oapi-codegen
- write tests against the generated handler

## Mental Model

Each API directory owns two user-written files:

```text
api/users/userid/
  frourio.go  # API contract: FrourioSpec, named types, FrourioPath
  route.go    # implementation: var Route = DefineRoute(RouteHandlers{...})
```

Generated files are glue and **must not be hand-edited**:

```text
api/users/userid/frourio_relay.go  # per-route types: GetRequest, GetStatus200, ...
api/frourio_server.go              # root net/http integration: Handler(), Mount()
api/openapi.json (or path from --openapi)
```

Change `frourio.go` or `route.go`, then regenerate.

## Basic Workflow

1. Find or create the target API directory under your `api/` root.
2. Edit or create `frourio.go` (the spec).
3. Edit or create `route.go` (the handler).
4. Run `go generate` to refresh `frourio_relay.go`, `frourio_server.go`, `openapi.json`.
5. Run tests.
6. If OpenAPI clients are part of the fixture, regenerate them too.

For this repository's basic test app:

```bash
go generate ./tests/apps/basic
go generate ./tests/apps/basic/openapiclient
go test ./...
```

The first command regenerates the frourio-go server and OpenAPI from
`tests/apps/basic/main.go`'s `go:generate` directive. The second command uses
oapi-codegen to regenerate the OpenAPI client fixture.

## Minimum Viable Endpoint

```go
// api/users/frourio.go
package users

type FrourioSpec struct {
    Get struct {
        Query struct {
            Limit *int `json:"limit" validate:"omitempty,min=1,max=100"`
        }
        Res struct {
            Status200 struct {
                Body []string `json:"body"`
            }
        }
    }
}
```

```go
// api/users/route.go
package users

import "context"

var Route = DefineRoute(RouteHandlers{
    Get: func(ctx context.Context, req GetRequest) (GetResponse, error) {
        return GetStatus200{Body: []string{"alice", "bob"}}, nil
    },
})
```

This produces `GET /api/users` with a JSON body response.

## Topic Index

Load the topic file for the area you are working on. Each is self-contained.

- [spec.md](spec.md) — `FrourioSpec` shape, supported HTTP methods, request parts, status codes, named types, comments-as-OpenAPI.
- [routing.md](routing.md) — directory-to-URL mapping, scalar `Param`, catch-all, optional catch-all, `FrourioPath` override, root restrictions.
- [request.md](request.md) — `Query`, `Header`, `Body`, JSON tag rules, supported scalar types, `URLEncoded` and `FormData` body markers.
- [response.md](response.md) — status code naming, `Body` content-type inference, `Header` writing, `Header.ContentType` override, multipart response with `FormData`, raw responses.
- [validation.md](validation.md) — go-playground/validator integration, request vs response validation, error response shape, OpenAPI integration.
- [middleware.md](middleware.md) — `Middleware.All`, method-specific middleware, typed `Context` vs `bool` placeholder, ancestor inheritance, `*http.Request` access, guard pattern.
- [openapi.md](openapi.md) — generated OpenAPI shape, `operationId`, schema naming, `x-go-validate` / `x-frourio-catch-all` / `x-go-type` extensions, default `FrourioError` schema.
- [generation.md](generation.md) — CLI (`generate` and `openapi` commands), flags, generated file layout, `go:generate` conventions, oapi-codegen integration.
- [testing.md](testing.md) — `httptest.Server` patterns, OpenAPI client tests, asserting JSON / multipart / raw bodies, middleware-driven status assertions.
- [pitfalls.md](pitfalls.md) — frequent mistakes, generator error messages, what to regenerate when things stop compiling.

## Server Startup

The root generated package exposes `Handler()` and `Mount(mux)`.

```go
package main

import (
    "log"
    "net/http"
    "time"

    "example.com/app/api"
)

func main() {
    server := &http.Server{
        Addr:              ":8080",
        Handler:           api.Handler(),
        ReadHeaderTimeout: 5 * time.Second,
    }
    log.Fatal(server.ListenAndServe())
}
```

Prefer `http.Server` with timeouts over `http.ListenAndServe` directly.

## Quick Lookup

- HTTP methods: `Get`, `Post`, `Put`, `Patch`, `Delete`, `Head`, `Options` — see [spec.md](spec.md).
- Request parts: `Param`, `Query`, `Header`, `Body` — see [request.md](request.md).
- Status codes: `StatusNNN` (any 100–599) — see [response.md](response.md).
- Body format markers: `URLEncoded bool`, `FormData bool` — see [request.md](request.md), [response.md](response.md).
- Path parameter shapes: `Param string|int`, `Param []string`, `Param *[]string`, optional `*string` — see [routing.md](routing.md).
- URL override: `const FrourioPath = "..."` — see [routing.md](routing.md).
- Raw escape hatch: omit `Res`, return `RawResponseFunc(...)` — see [response.md](response.md).
- Middleware: `Middleware.All` and per-method, with `Context` struct or `bool` — see [middleware.md](middleware.md).
- Generated file names: `frourio_relay.go` per route, `frourio_server.go` at api root — see [generation.md](generation.md).

## Common Pitfalls (one-liners)

- Never edit `frourio_relay.go`, `frourio_server.go`, or generated OpenAPI clients.
- After changing `frourio.go`, regenerate before fixing handler signatures.
- `FrourioPath` and `Param` are mutually exclusive in the same directory.
- Don't mix methods that have `Param` with methods that don't in one route directory.
- Path-parameter directories should be lowercase Go-friendly names like `userid`, not `[id]`.

See [pitfalls.md](pitfalls.md) for full list.
