# Routing

URL paths are derived from directory structure under the api root. There are
three controls that tweak this mapping:

1. **Directory layout** — every directory becomes a static URL segment.
2. **`Param` field** — turns a directory's segment into a path parameter.
3. **`FrourioPath` constant** — overrides a segment with arbitrary characters.

## Directory → URL

```text
api/                    → /api
api/users/              → /api/users
api/users/userid/       → /api/users/{userid}    (when Param is declared)
api/blog/slug/          → /api/blog/{slug...}    (catch-all)
api/files/path/         → /api/files  AND  /api/files/{path...}  (optional catch-all)
api/products/sale/      → /api/products/<override>   (FrourioPath)
api/secure/admin/users/ → /api/secure/admin/users
```

The api root can be any directory. The `go:generate` directive specifies it
(e.g. `go run ../../.. generate ./api`). All routes mount under that root's
package name as the first URL segment.

## Path Parameters

To make a directory's segment a path parameter, declare `Param` inside the
method block:

### Scalar (single segment)

```go
// api/users/userid/frourio.go
package userid

type FrourioSpec struct {
    Get struct {
        Param int `validate:"required"`
        Res struct {
            Status200 struct { Body string `json:"body"` }
        }
    }
}
```

URL: `GET /api/users/{userid}`. The parameter name is the directory's name.
Supported scalar types are `string` and `int`. Use a pointer (`*string`,
`*int`) to make the parameter optional — omitting the segment then produces
a separate route that does not match the parameter slot.

The handler receives the param at `req.Param` (not wrapped in a struct):

```go
Get: func(ctx context.Context, req GetRequest) (GetResponse, error) {
    return GetStatus200{Body: fmt.Sprintf("user:%d", req.Param)}, nil
}
```

Validation tags work on `Param` directly — `validate:"required,gte=1"` is
typical. Validation errors return 422.

### Catch-All (variadic)

```go
// api/blog/slug/frourio.go
type FrourioSpec struct {
    Get struct {
        Param []string `validate:"required"`
        Res struct {
            Status200 struct { Body string `json:"body"` }
        }
    }
}
```

URL: `GET /api/blog/{slug...}`. Matches one or more remaining segments and
captures them as `req.Param []string`.

```go
Get: func(ctx context.Context, req GetRequest) (GetResponse, error) {
    return GetStatus200{Body: strings.Join(req.Param, "/")}, nil
}
```

### Optional Catch-All

```go
// api/files/path/frourio.go
type FrourioSpec struct {
    Get struct {
        Param *[]string
        Res struct {
            Status200 struct { Body string `json:"body"` }
        }
    }
}
```

URL: matches both `GET /api/files` and `GET /api/files/{path...}`. The
generator registers two routes pointing at the same handler.

```go
Get: func(ctx context.Context, req GetRequest) (GetResponse, error) {
    if req.Param == nil {
        return GetStatus200{Body: "root"}, nil
    }
    return GetStatus200{Body: strings.Join(*req.Param, "/")}, nil
}
```

`req.Param == nil` → no segments. `*req.Param` → the captured segments.

## `FrourioPath` URL Override

Some URL segments can't be Go package names (multibyte characters, dashes,
symbols). Keep the directory Go-friendly and override the URL segment via a
top-level constant:

```go
// api/products/sale/frourio.go
package sale

const FrourioPath = "セール品"

type FrourioSpec struct {
    Get struct {
        Res struct {
            Status200 struct { Body string `json:"body"` }
        }
    }
}
```

URL: `GET /api/products/セール品`.

### Constraints

- Must be a `const` of type `string`.
- Value must not contain `/`. The generator errors out otherwise.
- Cannot be combined with `Param` in the same `frourio.go` — it's one or the
  other.

## Restrictions

### Param ↔ Non-Param Mixing

All methods in one route directory must agree on whether they take `Param`.
You cannot have `Get` with `Param int` and `Post` without it in the same
directory; the generator rejects that. Split into two directories if you need
both shapes.

### Root Directory

The api root (the directory passed to `frourio-go generate`) cannot have
`Param`. Path parameters require a parent segment.

### Common Naming

Use lowercase Go-friendly directory names for path-parameter segments. The
directory name becomes the param identifier in:

- The URL pattern (`{userid}`)
- The OpenAPI parameter name
- The Go field on `GetRequest` (capitalized: `req.Param`)

`userid` is conventional in this project. `userId`, `[id]`, `:id`, etc., are
not valid Go package names and must be avoided.

## Cross-Reference

- Field shapes inside method blocks: [request.md](request.md)
- Catch-all OpenAPI representation: [openapi.md](openapi.md)
- Common path-related errors: [pitfalls.md](pitfalls.md)
