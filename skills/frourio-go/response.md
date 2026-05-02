# Responses

Each method declares zero or more status entries inside `Res`. Each entry
generates a Go type that the handler returns.

```go
Res struct {
    Status200 struct { Body []string `json:"body"` }
    Status404 struct { Body string   `json:"body"` }
}

// Generated handler return types:
// type GetResponse interface { ... }   // method-level union
// type GetStatus200 struct{ Body []string }
// type GetStatus404 struct{ Body string }
```

Inside the handler:

```go
Get: func(ctx context.Context, req GetRequest) (GetResponse, error) {
    if notFound {
        return GetStatus404{Body: "not found"}, nil
    }
    return GetStatus200{Body: []string{"alice"}}, nil
}
```

## Status Code Naming

Any `StatusNNN` where `NNN` is a 3-digit code from 100 to 599 is recognized.
Common examples:

- `Status200` (OK)
- `Status201` (Created)
- `Status204` (No Content) — typically `struct{}` with no body
- `Status400` (Bad Request)
- `Status401` (Unauthorized)
- `Status403` (Forbidden)
- `Status404` (Not Found)
- `Status409` (Conflict)
- `Status422` (Unprocessable Entity)
- `Status500` (Internal Server Error)

422 is automatically returned for validation failures. You can still declare
your own `Status422` when you want to validate something the framework
doesn't.

## `Body` Content-Type Inference

The `Body` field's Go type determines the content type written:

| `Body` type                          | Content-Type                          |
|--------------------------------------|---------------------------------------|
| `string`                             | `text/plain; charset=utf-8`           |
| `[]byte`                             | `application/octet-stream`            |
| struct, named struct, slice, map, … | `application/json`                    |
| struct + `FormData bool`             | `multipart/form-data; boundary=…`     |

Pointers to any of the above are dereferenced. Nil bodies are written as the
JSON literal `null`.

```go
Res struct {
    Status200 struct { Body string }   // text/plain
}

Res struct {
    Status200 struct { Body []byte }   // application/octet-stream
}

Res struct {
    Status200 struct { Body MyStruct }   // application/json
}
```

## Response `Header`

A `Header` field on a status block writes response headers before
`WriteHeader`:

```go
Res struct {
    Status200 struct {
        Header struct {
            ETag string `json:"etag"`
        }
        Body []string `json:"body"`
    }
}
```

The handler sets header values when constructing the response:

```go
return GetStatus200{
    Header: struct{ ETag string `json:"etag"` }{ETag: "abc123"},
    Body:   []string{"alice"},
}, nil
```

The JSON tag controls the actual HTTP header name (use canonical case like
`etag` or `x-trace-id`). Without a JSON tag, the Go field name is used.

You can also use a named `Header` type for reuse — see the
`TextHeader` example in [tests/apps/basic/api/forms/frourio.go](../../tests/apps/basic/api/forms/frourio.go).

## Content-Type Override

Adding a field literally named `ContentType` to the response `Header`
suppresses the framework's auto-write of `Content-Type`. The handler then
controls it directly:

```go
type TextHeader struct {
    ContentType string
}

Res struct {
    Status200 struct {
        Header TextHeader
        Body   string `validate:"required"`
    }
}
```

```go
return GetStatus200{
    Header: TextHeader{ContentType: "text/custom"},
    Body:   "plain",
}, nil
```

Without this field, a `string` body would auto-write `text/plain`. With it,
the handler's value wins.

## Multipart Response (`FormData`)

A status block with both `FormData bool` and a struct `Body` writes the body
as `multipart/form-data`:

```go
type MultipartResponseBody struct {
    Name  string `json:"name"`
    Count int
}

Res struct {
    Status200 struct {
        FormData bool
        Body     MultipartResponseBody
    }
}
```

Each struct field becomes a form-data part using its JSON name (or Go field
name when no tag). The boundary is generated and included in the `Content-Type`.

## Raw Responses

Omit `Res` entirely to bypass the typed response system and write the response
yourself.

```go
type FrourioSpec struct {
    Get struct{}
}
```

```go
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
```

`GetResponse` in raw mode is the `RawResponse` interface:

```go
type RawResponse interface {
    WriteHTTP(w http.ResponseWriter, r *http.Request) error
}
```

`RawResponseFunc` is a typed function literal that satisfies that interface.
You can also implement a custom struct with a `WriteHTTP` method.

Use raw responses for:

- Streaming / chunked output
- File downloads with custom logic
- Multipart upload **requests** that need access to `*multipart.Reader`
  (since `FormData bool` only handles non-file fields)
- Server-Sent Events
- WebSocket upgrades

A handler returning `nil` (no error, no response) is treated as a server
error and produces 500. Always return either a status response or a raw
response.

## Response Validation

Each `StatusNNN` body is validated before writing. A failure logs and writes
500 (it is a server-side bug, not a client issue). See
[validation.md](validation.md).

## Cross-Reference

- Status entries inside `FrourioSpec`: [spec.md](spec.md)
- Response shape in OpenAPI: [openapi.md](openapi.md)
- Testing response patterns: [testing.md](testing.md)
