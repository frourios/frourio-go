# frourio-go

frourio-go is a code generator for building typed HTTP APIs with Go's standard
`net/http` server, `github.com/go-playground/validator/v10`, and OpenAPI.

You describe each endpoint with Go structs in `frourio.go`, write the business
logic in `route.go`, and let frourio-go generate the glue code that connects
your handlers to `http.ServeMux`, validation, response writing, and OpenAPI.

The goal is to keep API contracts close to the code that implements them while
still producing a portable OpenAPI document for clients, documentation, and CI.

## Quick Start With AI Coding Tools

This repo ships a Skill at [skills/frourio-go/](skills/frourio-go/) for AI
coding tools that support the Skill format. Point your tool at it and ask for
what you want — adding endpoints, editing specs, regenerating, writing tests.
The agent reads the Skill and follows the project's conventions.

## Design Principles

- Use Go types as the source of truth for request and response contracts.
- Keep route implementation files hand-written and easy to review.
- Generate only the repetitive HTTP glue: decoding, validation, routing, typed
  handler signatures, response writing, and OpenAPI.
- Use `net/http` instead of requiring a framework runtime.
- Treat OpenAPI 3.0.3 as a standard output, not an optional afterthought.
- Prefer stable generated names so external OpenAPI client generators produce
  predictable APIs.
- Avoid hidden global state. Middleware passes typed values through generated
  context structs instead of relying on `context.Value` for business data.

## What It Generates

For an API tree like this:

```text
api/
  frourio.go
  route.go
  users/
    userid/
      frourio.go
      route.go
```

frourio-go generates:

```text
api/
  frourio_relay.go       # typed request/response helpers for api/route.go
  frourio_server.go      # http.ServeMux registration for the whole API tree
  users/
    userid/
      frourio_relay.go   # typed helpers for users/userid/route.go
```

OpenAPI 3.0.3 document (`openapi.json`) is written only when `--openapi <path>`
is given (or via the `openapi` subcommand's `--output <path>`); without it the
relay/server files are generated and the OpenAPI step is skipped.

`route.go` is never generated. It is your application code.

## Endpoint Definition

Each endpoint directory contains a `frourio.go` file with a `FrourioSpec` type.
HTTP methods are represented by fields such as `Get`, `Post`, `Put`, `Patch`,
`Delete`, `Head`, and `Options`.

```go
package api

type FrourioSpec struct {
	Get struct {
		Query struct {
			Search *string `json:"search"`
			Limit  *int    `json:"limit" validate:"omitempty,min=1,max=100"`
		}
		Res struct {
			Status200 struct {
				Body []User `json:"body" validate:"required"`
			}
		}
	}
	Post struct {
		Body CreateUserBody
		Res struct {
			Status201 struct {
				Body User `json:"body" validate:"required"`
			}
			Status400 struct {
				Body string `json:"body"`
			}
		}
	}
}

type User struct {
	ID   int    `json:"id" validate:"required"`
	Name string `json:"name" validate:"required"`
}

type CreateUserBody struct {
	Name string `json:"name" validate:"required"`
}
```

The generated relay file gives `route.go` typed request and response values:

```go
package api

import "context"

var Route = DefineRoute(RouteHandlers{
	Get: func(ctx context.Context, req GetRequest) (GetResponse, error) {
		return GetStatus200{
			Body: []User{{ID: 1, Name: "Alice"}},
		}, nil
	},
	Post: func(ctx context.Context, req PostRequest) (PostResponse, error) {
		user := User{ID: 2, Name: req.Body.Name}
		return PostStatus201{Body: user}, nil
	},
})
```

## Mounting The Server

frourio-go generates one `frourio_server.go` at the API root. It exposes
`Mount` and `Handler`, so the generated API can be attached to a standard Go
HTTP server.

```go
package main

import (
	"log"
	"net/http"

	"example.com/app/api"
)

func main() {
	log.Fatal(http.ListenAndServe(":8080", api.Handler()))
}
```

`api.Handler()` returns a prefix-agnostic `http.Handler`. If you want to mount
it under a URL prefix like `/api`, compose it with `http.StripPrefix` in your
`main.go` — frourio-go does not bake any prefix into the generated routes.

```go
mux := http.NewServeMux()
mux.Handle("/api/", http.StripPrefix("/api", api.Handler()))
log.Fatal(http.ListenAndServe(":8080", mux))
```

## Routing Model

Routes are derived from directories. A directory is a static URL segment unless
its `FrourioSpec` defines `Param`.

```text
api/frourio.go                         -> /
api/users/userid/frourio.go            -> /users/userid
api/products/sale/frourio.go           -> /products/sale
```

With `Param`, the final directory segment becomes a path parameter:

```go
package userid

type FrourioSpec struct {
	Get struct {
		Param int `validate:"required"`
		Res struct {
			Status200 struct {
				Body string `json:"body"`
			}
		}
	}
}
```

This produces `/users/{userid}` and passes `Param int` to the handler.

Catch-all parameters are inferred from the `Param` type:

```go
Param []string  // /blog/{slug...}
Param *[]string // /blog and /blog/{slug...}
```

If the URL segment needs characters that are awkward or invalid in Go import
paths, keep the directory Go-friendly and override the URL segment with
`FrourioPath`.

```go
package sale

const FrourioPath = "セール品"
```

## Validation And Decoding

Request validation uses `github.com/go-playground/validator/v10`.

frourio-go decodes and validates:

- path parameters
- query parameters
- headers
- JSON request bodies
- `application/x-www-form-urlencoded` bodies
- `multipart/form-data` non-file fields

Form body formats are selected with marker fields:

```go
type FrourioSpec struct {
	Post struct {
		URLEncoded bool
		Body struct {
			Name string `json:"name" validate:"required"`
			Age  int    `json:"age" validate:"gte=1"`
		}
		Res struct {
			Status201 struct {
				Body string `json:"body"`
			}
		}
	}
	Put struct {
		FormData bool
		Body struct {
			Title string `json:"title" validate:"required"`
		}
		Res struct {
			Status200 struct {
				Body string `json:"body"`
			}
		}
	}
}
```

Query and form values are converted to their Go types before validation, so a
field like `Limit *int` receives an integer, not a raw string.

## Responses

Response variants are declared under `Res` with `StatusXXX` fields.

```go
Res struct {
	Status200 struct {
		Body User `json:"body"`
	}
	Status404 struct {
		Body string `json:"body"`
	}
}
```

Response body handling is inferred from the `Body` type:

- `struct`, named struct, slices, maps, and most Go values are written as JSON.
- `string` is written as `text/plain`.
- `[]byte` is written as `application/octet-stream`.
- `FormData bool` with a struct body is written as `multipart/form-data`.
- If a response header defines `ContentType`, frourio-go does not overwrite it.
- If `Res` is omitted, the route is treated as a raw response escape hatch.

## Typed Middleware

Middleware is declared in `FrourioSpec.Middleware` and implemented in
`RouteHandlers.Middleware`.

`All` applies to all methods in the route. Method-specific middleware such as
`Get` or `Post` applies only to that method.

```go
package api

type FrourioSpec struct {
	Middleware struct {
		All struct {
			Context struct {
				TraceID string `json:"traceId" validate:"required"`
			}
		}
		Get struct {
			Context struct {
				ReadScope string `json:"readScope" validate:"required"`
			}
		}
	}
	Get struct {
		Res struct {
			Status200 struct {
				Body string `json:"body"`
			}
		}
	}
}
```

Implementation:

```go
package api

import "context"

var Route = DefineRoute(RouteHandlers{
	Middleware: RouteMiddleware{
		All: func(ctx context.Context, next MiddlewareNext) (any, error) {
			return next(ctx, MiddlewareAllContext{TraceID: "trace-1"})
		},
		Get: func(ctx context.Context, req GetRequest, mw MiddlewareContext, next GetNext) (GetResponse, error) {
			return next(ctx, req, GetMiddlewareContext{ReadScope: "users:read"})
		},
	},
	Get: func(ctx context.Context, req GetRequest, mw GetContext) (GetResponse, error) {
		return GetStatus200{Body: mw.TraceID + ":" + mw.ReadScope}, nil
	},
})
```

Use `Middleware.All bool` or `Middleware.Get bool` for middleware that does not
add typed context values.

## OpenAPI

OpenAPI generation is a core part of frourio-go. The generated document includes:

- stable `operationId` values
- path, query, and header parameters
- request bodies
- status-code-specific responses
- reusable `components.schemas`
- validator-derived schema constraints where supported
- `summary` and `description` from comments

Comment rules are intentionally simple:

```go
type FrourioSpec struct {
	// List users
	/*
		Returns users visible to the current caller.
			Indented lines are preserved in the description.
	*/
	Get struct {
		Query struct {
			// Search term
			Search *string `json:"search"`
		}
	}
}
```

Line comments become `summary`. Block comments become `description`.

frourio-go does not generate a proprietary HTTP client. The OpenAPI document is
the integration point for Go, TypeScript, Java, Kotlin, Swift, Python, C#, and
other ecosystems through existing OpenAPI client generators.

## CLI

```bash
frourio-go generate ./api
frourio-go generate ./api --openapi ./openapi.json
frourio-go generate ./api --openapi ./openapi.json --template ./openapi_template.json
frourio-go openapi   ./api --output ./openapi.json
frourio-go openapi   ./api --output ./openapi.json --template ./openapi_template.json
```

`generate` writes relay files and the root server file. The OpenAPI document
is written only when `--openapi <path>` is supplied. `openapi` writes only the
OpenAPI document and requires `--output <path>`.

### OpenAPI template

When OpenAPI is generated, frourio-go reads a JSON template and merges its
fields into the output. This is where you put settings that are outside
frourio-go's responsibility — `info`, `servers`, `tags`, `security`,
`externalDocs`, custom schemas, etc.

- `--template <path>` lets you point at any file. Missing → error.
- Without `--template`, the default is `openapi_template.json` next to the
  output file. Missing → frourio-go writes a minimal skeleton there
  (`openapi`, `info.title`, `info.version`) and uses it.
- Merge rules: template fields are kept; the generator's `paths` and
  `components.schemas` always override anything the template tries to put
  under those slots (frourio-go owns the API contract). Custom schemas you
  add under `components.schemas.<YourType>` are preserved as long as the
  name doesn't collide with a generated one.

