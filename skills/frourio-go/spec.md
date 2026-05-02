# `frourio.go` — Spec File

Each route directory has one `frourio.go` defining a `FrourioSpec` type. This is
the source of truth for request/response shapes, validation, and OpenAPI.

## Required Identifier

```go
package users

type FrourioSpec struct {
    // ... methods and Middleware go here
}
```

The type name must be exactly `FrourioSpec`. The package name follows Go
conventions and matches the directory name (must be a valid Go identifier — see
[routing.md](routing.md) for `FrourioPath` workaround when the URL needs symbols
or multibyte characters).

## Supported HTTP Methods

Add a struct field per method you want to expose. Field names are matched
case-sensitively against this list:

- `Get`
- `Post`
- `Put`
- `Patch`
- `Delete`
- `Head`
- `Options`

```go
type FrourioSpec struct {
    Get struct { /* ... */ }
    Post struct { /* ... */ }
    Delete struct { /* ... */ }
}
```

Methods absent from the spec respond with `404`. Methods present but with no
handler in `route.go` will fail to compile.

## Method Block Structure

Each method block can have:

| Field        | Purpose                                                       |
|--------------|---------------------------------------------------------------|
| `Param`      | Single path parameter (only in `{name}` directories)          |
| `Query`      | URL query parameters                                          |
| `Header`     | Request headers                                               |
| `Body`       | Request body (JSON by default, or with `URLEncoded`/`FormData`) |
| `URLEncoded` | Marker `bool`: decode body as `application/x-www-form-urlencoded` |
| `FormData`   | Marker `bool`: decode body as `multipart/form-data` non-file fields |
| `Res`        | Response declarations: `StatusNNN` blocks                     |

Omit `Res` entirely to declare a **raw response** method — see
[response.md](response.md#raw-responses).

```go
Get struct {
    Param int `validate:"required"`
    Query struct {
        Limit *int `json:"limit"`
    }
    Header struct {
        XAPIKey string `json:"x-api-key" validate:"required"`
    }
    Res struct {
        Status200 struct { /* ... */ }
        Status404 struct { /* ... */ }
    }
}
```

## Status Codes

Each entry in `Res` is named `Status` followed by a 3-digit code (100–599):

```go
Res struct {
    Status200 struct { Body string }
    Status201 struct { Body string }
    Status204 struct{} // no body
    Status400 struct { Body string }
    Status404 struct { Body string }
    Status500 struct { Body string }
}
```

Each declared `StatusNNN` becomes a concrete Go type returned from the handler
(e.g. `GetStatus200`, `PostStatus404`). The handler return type is the method's
union interface (`GetResponse`, `PostResponse`, …).

`Status204` with `struct{}` legitimately writes 204 No Content with no body.

## Named Types

Define request/response types as named types in the same `frourio.go` file
when the structure is reused or non-trivial:

```go
type FormPostBody struct {
    Name   string    `json:"name" validate:"required"`
    Age    int       `json:"age" validate:"gte=1"`
    Scores []float64 `json:"score" validate:"required"`
}

type TextHeader struct {
    ContentType string
}

type FrourioSpec struct {
    Get struct {
        Res struct {
            Status200 struct {
                Header TextHeader
                Body   string `validate:"required"`
            }
        }
    }
    Post struct {
        URLEncoded bool
        Body       FormPostBody
        Res struct {
            Status201 struct {
                Body string `json:"body" validate:"required"`
            }
        }
    }
}
```

Named types are picked up automatically and used as-is in OpenAPI schemas.
You can also import named types from another package (typically a parent or
sibling under `api/`) for cross-route reuse.

## Comments → OpenAPI

Comments **directly above** a spec element are propagated to OpenAPI:

- Line comments (`// ...`) become `summary`.
- Block comments (`/* ... */`) become `description`.
- Using both gives both fields on the same operation/parameter/property.

```go
type FrourioSpec struct {
    // List users
    /*
        Returns users visible to the current caller.
            Indented details are preserved.
    */
    Get struct {
        Query struct {
            // Search term
            Search *string `json:"search"`
            /*
                Maximum number of items.
                    Indented limit detail.
            */
            Limit *int `json:"limit"`
        }
        Res struct {
            // Successful root response
            Status200 struct {
                Body []string `json:"body"`
            }
        }
    }
}
```

Block comment indentation is normalized: gofmt-inserted leading tabs are
stripped while user-intended nested indentation is preserved.

Comments work on:

- The whole `FrourioSpec` (treated as the path/tag context)
- Each method (`Get`, `Post`, …)
- Each `StatusNNN`
- Each field inside `Query`, `Body`, `Header`, `Param`
- Top-level named types (`type FormPostBody struct {...}`) — comments become
  the schema description in OpenAPI.

## What `FrourioSpec` Does Not Hold

- The URL path. That is derived from directory structure plus optional
  `FrourioPath` constant. See [routing.md](routing.md).
- The handler. That lives in `route.go`. See [request.md](request.md) and
  [response.md](response.md).
- Middleware bodies. The `Middleware` field declares the *shape* of the
  injected context; the implementation lives in `route.go`. See
  [middleware.md](middleware.md).

## Cross-Reference

- Path mapping and `Param`: [routing.md](routing.md)
- Field types and JSON tag rules: [request.md](request.md)
- Status code body shapes and content types: [response.md](response.md)
- `Middleware` declarations: [middleware.md](middleware.md)
