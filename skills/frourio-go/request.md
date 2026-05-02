# Request Parts

A method's request data has up to four named parts: `Param`, `Query`,
`Header`, `Body`. All are optional. They appear on the generated
`<Method>Request` struct passed to the handler.

```go
type FrourioSpec struct {
    Post struct {
        Param int `validate:"required"`
        Query struct {
            Filter *string `json:"filter"`
        }
        Header struct {
            XAPIKey string `json:"x-api-key" validate:"required"`
        }
        Body struct {
            Name string `json:"name" validate:"required"`
        }
    }
}

// Generated:
// type PostRequest struct {
//     Param  int
//     Query  struct{ Filter *string }
//     Header struct{ XAPIKey string }
//     Body   struct{ Name string }
// }
```

The handler signature:

```go
Post: func(ctx context.Context, req PostRequest) (PostResponse, error) {
    // req.Param, req.Query.Filter, req.Header.XAPIKey, req.Body.Name
}
```

(With middleware, an additional `mw <Method>Context` argument follows; see
[middleware.md](middleware.md).)

## `Param`

See [routing.md](routing.md) for shapes, optional, catch-all variants.
Briefly: scalar `string|int`, optional `*string|*int`, catch-all `[]string`,
optional catch-all `*[]string`.

## `Query`

URL query parameters. Either a struct literal or a named struct.

```go
Query struct {
    Search  *string   `json:"search"`
    Limit   *int      `json:"limit"`
    Active  *bool     `json:"active"`
    Scores  []float64 `json:"score"`
    RawName *string   // no json tag → field name used as the URL key
}
```

Decoded from `r.URL.Query()`. Multi-value entries (`?score=1&score=2`) bind to
slice fields.

### JSON Tag Behavior

| Tag                       | URL key used         |
|---------------------------|----------------------|
| `json:"search"`           | `search`             |
| `json:"search,omitempty"` | `search` (options ignored) |
| (no `json` tag)           | the Go field name (`RawName`, not lowercased) |
| `json:"-"`                | the field is hidden from JSON, but the decoder still tries to read its lowercased name |

### Required vs Optional

- Pointer types (`*string`, `*int`, `*bool`, `*float64`, …) are **optional**
  and stay nil when absent.
- Non-pointer scalars are **required**.
- Slice types (`[]string`, `[]float64`, …) are zero-length slices when absent.

To enforce presence on a non-pointer field, add `validate:"required"`. Since
zero values are valid Go values, `required` validates that the user provided
at least one query value (the typed value parsed cleanly).

### Supported Scalar Types

`string`, `bool`, `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`,
`uint16`, `uint32`, `uint64`, `float32`, `float64`. The same list applies
inside body forms. Decoding goes through `strconv.Parse*`. Invalid input
returns 422 with a per-field error.

## `Header`

Request headers. Same rules as `Query` for tags and types.

```go
Header struct {
    XAPIKey string `json:"x-api-key" validate:"required"`
    Trace   *string `json:"x-trace-id"`
}
```

Header lookups go through `r.Header.Get(name)` — the JSON tag is the actual
HTTP header name (case-insensitive on the wire, but write canonical-case in
the tag).

## `Body`

Default body format is JSON. Two marker fields switch it to form formats.

### JSON (default)

```go
Body struct {
    Name string `json:"name" validate:"required"`
    Age  *int   `json:"age"`
}
```

Decoded with `json.NewDecoder`. Validates the whole struct after decode.

### URL-encoded form

```go
Post struct {
    URLEncoded bool         // marker
    Body struct {
        Name   string `json:"name" validate:"required"`
        Active bool   `json:"active"`
    }
    Res struct { /* ... */ }
}
```

`URLEncoded bool` is a directive — its value is never read. Decoded from
`r.PostForm`. Same scalar type set as `Query`. Multi-value fields bind to
slices.

### Multipart form-data

```go
Put struct {
    FormData bool          // marker
    Body struct {
        Title string `json:"title" validate:"required"`
        Count uint8  `json:"count" validate:"gte=1"`
    }
    Res struct { /* ... */ }
}
```

`FormData bool` is a directive. Decoded from non-file fields of multipart
form-data. The runtime calls `ParseMultipartForm(32 << 20)` (32 MB). For file
uploads, use a [raw response](response.md#raw-responses) handler that reads
from the request directly instead.

`URLEncoded` and `FormData` are mutually exclusive in one method block.

## Validation

Validation tags use `github.com/go-playground/validator/v10`. They work on:

- `Param` itself (`Param int \`validate:"required"\``)
- Each field of `Query`, `Header`, `Body`

Validation runs **after decode and before middleware/handler**. Failures
return 422 with a structured error (see [validation.md](validation.md)).

## Untagged Fields & `body` Quirk

A field without any JSON tag uses the Go field name as-is for JSON, query, and
form decoding (capitalization preserved). This is useful when the Go field
name is already the desired key — including unexported lowercase names like
`body` (some test fixtures use this; the type lookup still works, but
exporting is the safer default).

## Real Examples in This Repo

- Mixed `Query` types: [tests/apps/basic/api/frourio.go](../../tests/apps/basic/api/frourio.go)
- JSON `Body` with optional field: [tests/apps/basic/api/users/frourio.go](../../tests/apps/basic/api/users/frourio.go)
- `URLEncoded` with named body type: [tests/apps/basic/api/forms/frourio.go](../../tests/apps/basic/api/forms/frourio.go)
- `FormData` request: same `forms` directory
- `Header` injection: [tests/apps/middleware/api/mw/route.go](../../tests/apps/middleware/api/mw/route.go) reads `Authorization` and `X-Trace-Id` directly off `*http.Request`

## Cross-Reference

- Status codes and response bodies: [response.md](response.md)
- Validator tag reference and error shape: [validation.md](validation.md)
- Path parameters: [routing.md](routing.md)
