# OpenAPI Generation

frourio-go can produce an OpenAPI 3.0.3 document alongside the Go relay and
server files. Output is **opt-in** — you must supply a path; there is no
default location.

## When It's Generated

- `frourio-go generate <api-dir> --openapi <path> [--template <path>]` —
  generates Go relay / server files **and** writes OpenAPI to `<path>`.
  Without `--openapi`, only the Go files are produced.
- `frourio-go openapi <api-dir> --output <path> [--template <path>]` —
  generates **only** the OpenAPI document, without touching Go files.
  `--output` is required. Useful for CI checks or doc-only updates.

## Template Merging

Settings that are outside frourio-go's responsibility — `info`, `servers`,
`tags`, `security`, `externalDocs`, custom schemas — go in a JSON template
that frourio-go reads on every generation and merges into the output.

| Source            | Wins for                                            |
|-------------------|-----------------------------------------------------|
| Template          | `info`, `servers`, `tags`, `security`, `externalDocs`, anything generator doesn't emit |
| Template          | `components.schemas.<YourType>` for names not collided with generated ones |
| Generator         | `paths` (always — generator owns the API contract)  |
| Generator         | `components.schemas.<GeneratedType>` for any name it emits |

Resolution:

- `--template <path>` — explicit path. Missing file is an error.
- No `--template` — defaults to `openapi_template.json` next to the OpenAPI
  output. Missing → frourio-go writes a minimal skeleton there
  (`openapi: 3.0.3`, `info.title`, `info.version`) and uses it.

Commit the template alongside the source. It's a hand-edited input, not a
generated artifact, even when frourio-go scaffolds the initial skeleton.

## What's in the Document

### Top-level

- `openapi`: `3.0.3` (fixed)
- `info.title`: `frourio-go API`
- `info.version`: `0.1.0`
- `paths`: one entry per discovered route
- `components.schemas`: named types from `frourio.go` files plus the
  framework-supplied `FrourioError`

### Paths

Each `frourio.go` method becomes one operation under the route's URL:

```text
api/users/userid/   →  /users/{userid}    (GET, etc.)
api/blog/slug/      →  /blog/{slug...}
api/files/path/     →  /files  AND  /files/{path...}   (optional catch-all)
```

The OpenAPI document reflects the prefix-agnostic routing — paths begin with
the api root (`/`), not `/api`. Add a `servers` entry in the template to point
clients at the deployed prefix (e.g. `https://example.com/api`).

Optional catch-all routes are emitted as **two** path entries — the base
path and the catch-all — both bound to the same operation pair.

### Operation IDs

Auto-generated as `<method><PascalCasePath>`:

| Path                          | Method | operationId             |
|-------------------------------|--------|-------------------------|
| `/users`                      | GET    | `getUsers`              |
| `/users`                      | POST   | `postUsers`             |
| `/users/{userid}`             | GET    | `getUsersByUserid`      |
| `/blog/{slug...}`             | GET    | `getBlogBySlug`         |

Hyphens, underscores, and dots in path segments become PascalCase boundaries.
Path parameters become `By<ParamName>`.

### Parameters

`Param`, `Query`, and `Header` fields become entries in the operation's
`parameters` array. Path parameters cascade — a route under
`api/users/userid/posts/postid/` emits one `path` parameter per ancestor
(`userid`) plus its own (`postid`). Each parameter carries:

- `name` (from JSON tag, falling back to Go field name)
- `in` (`path`, `query`, or `header`)
- `required` (true unless the Go type is a pointer)
- `schema` (mapped from the Go type)
- `description` (from line comment above the field)

Catch-all parameters add `style: simple`, `explode: false`, and the
extension `x-frourio-catch-all: true`.

### Request Body

When `Body` is declared, the operation's `requestBody` reflects the format:

| Marker          | Content-Type                          |
|-----------------|---------------------------------------|
| (default)       | `application/json`                    |
| `URLEncoded bool` | `application/x-www-form-urlencoded` |
| `FormData bool` | `multipart/form-data`                 |

`required: true` on the request body itself. Body field validation is on the
schema (see below).

### Responses

Each declared `StatusNNN` becomes a response object:

- `description` from a comment above the status block, or a default like
  `"Successful response"` / `"Bad Request"`.
- `content` matches the body's inferred content-type (see
  [response.md](response.md)).
- `headers` for any fields on the status block's `Header`.

A `422 Unprocessable Entity` response is **always** added, referencing the
built-in `FrourioError` schema.

### `FrourioError` Schema

Predefined under `components.schemas.FrourioError`:

```yaml
type: object
properties:
  status:
    type: integer
  error:
    type: string
  issues:
    type: array
    items:
      type: object
      properties:
        path:    { type: array, items: {} }   # heterogeneous path segments
        message: { type: string }
```

`path` is an array of segments — typically `["body"]`, `["body", "field"]`,
`["query", "field"]`, `["param"]`, etc. Clients generated by oapi-codegen
receive a typed binding for this.

## Schema Naming

For inline structs declared inside `FrourioSpec`, the generator builds names
from the route + method + status + part:

| Source                                         | Schema name                |
|-----------------------------------------------|----------------------------|
| `Get.Res.Status200.Body` inline struct         | `GetStatus200Body`         |
| `Post.Body` inline struct in `api/users/`      | `PostApiUsersBody`         |
| `Get.Res.Status200.Header` inline struct       | `GetStatus200Header`       |

For named types in the same `frourio.go` (e.g. `FormPostBody`), the generator
uses the Go type name directly. Comments on the type definition become the
schema's `description`.

## Type Mapping

| Go type                                                                                | OpenAPI schema                               |
|----------------------------------------------------------------------------------------|----------------------------------------------|
| `string`                                                                               | `type: string`                               |
| `int`, `int64`, `uint32`, …                                                            | `type: integer`                              |
| `float32`, `float64`                                                                   | `type: number`                               |
| `bool`                                                                                 | `type: boolean`                              |
| `[]T`                                                                                  | `type: array`, `items:` (T's schema)         |
| `map[string]T`                                                                         | `type: object`, `additionalProperties:` (T's schema) |
| struct                                                                                 | `type: object` with `properties` and `required` |
| `*T`                                                                                   | T's schema (pointer indicates optional)     |
| Non-primitive named Go types (e.g. `time.Time`)                                       | best-effort + `x-go-type` extension          |

## Extensions Used

- `x-frourio-catch-all: true` — catch-all path parameter.
- `x-go-validate: "<tag value>"` — verbatim go-playground/validator rule
  string preserved on the property's schema. The generator does **not** map
  these to native OpenAPI keywords; consumers that want to enforce the same
  rules read the extension.
- `x-go-type: "<Go type>"` — when a property's type isn't a clean OpenAPI
  primitive, the original Go type name is recorded for client generators
  that understand it.

## OpenAPI-Only Comments

Comments in `frourio.go` directly above a spec element flow through:

- Above a method (`Get`, `Post`, …): operation `summary` and `description`.
- Above a `StatusNNN`: response `description`.
- Above a field of `Query`, `Body`, `Param`, `Header`: parameter / property
  `description`.
- Above a top-level named type (`type FormPostBody struct {...}`): schema
  `description`.

See [spec.md](spec.md) for the line-vs-block-comment rules.

## Output Location

```bash
go run ../../.. generate ./api                       # no OpenAPI written
go run ../../.. generate ./api --openapi ../doc.json # → ../doc.json
```

The basic test app passes `--openapi ./openapi.json` so the sibling
`openapiclient` package can consume the doc. See
[tests/apps/basic/main.go](../../tests/apps/basic/main.go) and
[tests/apps/basic/openapi.json](../../tests/apps/basic/openapi.json).

## Cross-Reference

- Generation CLI / flags: [generation.md](generation.md)
- oapi-codegen integration: [generation.md](generation.md#openapi-client-generation)
- Validation tag mapping: [validation.md](validation.md)
