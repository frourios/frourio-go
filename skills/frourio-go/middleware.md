# Middleware

Middleware in frourio-go has two parts:

1. **Spec** — `FrourioSpec.Middleware` declares the *shape* of typed
   per-request context the middleware injects, plus which methods get a
   per-method middleware.
2. **Implementation** — `RouteMiddleware` in `route.go` provides the actual
   middleware functions.

Both are typed against generated names (`MiddlewareNext`, `MiddlewareAllContext`,
`<Method>MiddlewareContext`, `<Method>Next`, `<Method>Context`).

## Two Granularities

```go
type FrourioSpec struct {
    Middleware struct {
        // Runs for every method on this route AND descendants
        All struct {
            Context struct {
                TraceID string `json:"traceId" validate:"required"`
            }
        }
        // Runs only for POST on this route (after All)
        Post struct {
            Context struct {
                Role string `json:"role" validate:"required"`
            }
        }
    }
    Get struct { /* ... */ }
    Post struct { /* ... */ }
}
```

- `All` runs once per request before the method handler. Its context is
  visible to all handlers in this route and any descendant routes.
- A method-specific middleware (e.g. `Post`) runs after `All`, only for that
  method on this route. Its context is visible only inside that method's
  handler.

A method-specific middleware in the spec requires the matching method in
the spec too — declaring `Middleware.Post` without `FrourioSpec.Post` is a
generation-time error.

## Spec → Generated Types

For `Middleware.All` with a `Context` struct, the generator emits:

- `MiddlewareAllContext` — concrete struct with the fields you declared
- `MiddlewareContext` — same fields plus inherited ancestor fields (used by
  method-specific middleware)
- `MiddlewareNext` — the function to advance the chain

For `Middleware.<Method>` with a `Context` struct, the generator emits:

- `<Method>MiddlewareContext` — fields declared on this method's `Context`
- `<Method>Next` — function to call to advance
- `<Method>Context` — what the *handler* receives: union of all ancestor
  `All` contexts + this route's `All` context + this method's `Context`

## Implementing in `route.go`

### `All` with typed context

```go
import (
    "context"
    "net/http"
)

var Route = DefineRoute(RouteHandlers{
    Middleware: RouteMiddleware{
        All: func(ctx context.Context, r *http.Request, next MiddlewareNext) (any, error) {
            return next(ctx, MiddlewareAllContext{TraceID: "trace-123"})
        },
    },
    Get: func(ctx context.Context, req GetRequest, mw GetContext) (GetResponse, error) {
        return GetStatus200{Body: mw.TraceID}, nil
    },
})
```

`*http.Request` is available for inspecting headers, cookies, remote addr,
etc.

### Method-specific middleware

```go
var Route = DefineRoute(RouteHandlers{
    Middleware: RouteMiddleware{
        All: func(ctx context.Context, r *http.Request, next MiddlewareNext) (any, error) {
            return next(ctx, MiddlewareAllContext{
                IsAdmin:     true,
                Permissions: []string{"read", "write", "delete"},
            })
        },
        Post: func(ctx context.Context, r *http.Request, req PostRequest, mw MiddlewareContext, next PostNext) (PostResponse, error) {
            if !mw.IsAdmin {
                return PostStatus403{Body: "Forbidden: Admin access required"}, nil
            }
            return next(ctx, req)
        },
    },
    Post: func(ctx context.Context, req PostRequest, mw PostContext) (PostResponse, error) {
        return PostStatus201{Body: "..."}, nil
    },
})
```

Method middleware can:

- **Short-circuit** — return a typed `PostResponseN` directly (the handler is
  skipped).
- **Continue** — call `next(ctx, req)` (or `next(ctx, req, PostMiddlewareContext{...})`
  if the method declares a `Context`).

The `mw MiddlewareContext` parameter exposes ancestor + current `All` context
fields. The `mw PostContext` on the handler is the merge of all ancestor
contexts plus this method's middleware context.

## `bool` Marker (No Typed Context)

When a middleware doesn't need to inject context fields, declare it as a
plain `bool` instead of a struct with `Context`:

```go
type FrourioSpec struct {
    Middleware struct {
        All bool   // route-wide middleware, no typed context
        Post bool  // method-specific middleware, no typed context
    }
}
```

This still requires the matching `RouteMiddleware.All` / `RouteMiddleware.Post`
implementation, but the generated `Next` function omits the context argument.

The most common reason to use the `bool` form: the middleware writes context
through `context.WithValue`, observes `*http.Request`, or short-circuits with
a response — but doesn't need to expose typed values to the handler.

## Ancestor Inheritance

Every route's `Middleware.All` runs on all descendants. The order is
ancestor-first:

```text
api/secure/                 → All sets {UserID, TraceID}
api/secure/admin/           → All sets {IsAdmin, Permissions} on top of inherited
api/secure/admin/users/     → handler sees {UserID, TraceID, IsAdmin, Permissions}
```

```go
// api/secure/admin/users/route.go
Get: func(ctx context.Context, req GetRequest, mw GetContext) (GetResponse, error) {
    // mw.UserID, mw.TraceID, mw.IsAdmin, mw.Permissions all visible
}
```

A descendant must declare a `FrourioSpec` (even an empty one with a method)
to participate. There is no opt-out for inherited middleware: if you need a
sibling route that bypasses a parent's middleware, place it outside that
parent's directory.

For example, in this repo, the `middleware` test app uses
`api/public/` (sibling of `api/mw/`) rather than `api/mw/public/`, because
the latter would inherit `api/mw/`'s middleware.

## Pure Middleware Routes

A route that only provides middleware to descendants doesn't need handler
methods. Declare `FrourioSpec` with only `Middleware`:

```go
type FrourioSpec struct {
    Middleware struct {
        All struct {
            Context struct {
                TraceID string `json:"traceId" validate:"required"`
            }
        }
    }
}
```

```go
var Route = DefineRoute(RouteHandlers{
    Middleware: RouteMiddleware{
        All: func(ctx context.Context, r *http.Request, next MiddlewareNext) (any, error) {
            return next(ctx, MiddlewareAllContext{TraceID: "nested-trace"})
        },
    },
})
```

This produces no HTTP endpoint at this directory's URL, but its middleware
applies to all descendants. See
[tests/apps/basic/api/nest/](../../tests/apps/basic/api/nest/) for a working
example.

## Reading from `*http.Request`

The `r *http.Request` parameter gives middleware full access to:

- `r.Header.Get("Authorization")` — auth headers
- `r.Cookie("...")` — session cookies
- `r.RemoteAddr` — client IP (subject to proxy headers)
- `r.URL.Query()` — raw query (typed `Query` is also available on the
  inner request struct in method-specific middleware)
- Any other standard `net/http` field

A worked example reading Bearer tokens and trace headers is at
[tests/apps/middleware/api/mw/route.go](../../tests/apps/middleware/api/mw/route.go).

## Validation of Middleware Context

Each `MiddlewareAllContext` and `<Method>MiddlewareContext` is validated by
the runtime after the middleware sets it. Required fields you forget to set
become 500 errors.

```go
Context struct {
    UserID string `json:"userId" validate:"required"`
}

// If middleware returns next(ctx, MiddlewareAllContext{}) → 500
```

## Testing Middleware

Drive tests through `httptest`:

- Unauthenticated request: omit `Authorization` and assert the 401/403
  response your method-specific middleware returns.
- Authenticated request: set `Authorization: Bearer ...` and assert the
  handler's success response.
- Inherited context: hit a deep child route and assert that fields set in
  ancestors are reflected in the body.

See [testing.md](testing.md).

## Cross-Reference

- Spec field shapes: [spec.md](spec.md)
- Where validation kicks in: [validation.md](validation.md)
- Working examples: [tests/apps/basic/api/auth/](../../tests/apps/basic/api/auth/),
  [tests/apps/basic/api/secure/admin/](../../tests/apps/basic/api/secure/admin/),
  [tests/apps/middleware/api/mw/](../../tests/apps/middleware/api/mw/)
