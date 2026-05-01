package generator

import (
	"encoding/json"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateBasicApp(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.26\n\nrequire github.com/go-playground/validator/v10 v10.30.2\n")
	api := filepath.Join(dir, "api")

	writeFile(t, api, "frourio.go", `package api

type FrourioSpec struct {
	Get struct {
		Header struct {
			RequestID string `+"`json:\"x-request-id\"`"+`
		}
		Query struct {
			Search *string `+"`json:\"search\"`"+`
			Limit  *int    `+"`json:\"limit\"`"+`
			Tags   []string `+"`json:\"tag\"`"+`
			Ids    []int    `+"`json:\"id\"`"+`
		}
		Res struct {
			Status200 struct {
				Body string `+"`json:\"body\" validate:\"required\"`"+`
				Header struct {
					ETag string `+"`json:\"etag\"`"+`
				}
			}
		}
	}
	Post struct {
		URLEncoded bool
		Body struct {
			Name string `+"`json:\"name\" validate:\"required\"`"+`
		}
		Res struct {
			Status201 struct {
				Body string `+"`json:\"body\" validate:\"required\"`"+`
			}
			Status400 struct{}
		}
	}
	Put struct {
		FormData bool
		Body struct {
			Title string `+"`json:\"title\" validate:\"required\"`"+`
			Count uint8 `+"`json:\"count\"`"+`
		}
		Res struct {
			Status200 struct {
				Body string `+"`json:\"body\" validate:\"required\"`"+`
			}
		}
	}
}
`)
	writeFile(t, api, "route.go", `package api

import "context"

var Route = DefineRoute(RouteHandlers{
	Get: func(ctx context.Context, req GetRequest) (GetResponse, error) {
		return GetStatus200{Body: "ok"}, nil
	},
	Post: func(ctx context.Context, req PostRequest) (PostResponse, error) {
		return PostStatus201{Body: req.Body.Name}, nil
	},
	Put: func(ctx context.Context, req PutRequest) (PutResponse, error) {
		return PutStatus200{Body: req.Body.Title}, nil
	},
})
`)
	writeFile(t, api, "users/userid/frourio.go", `package userid

type FrourioSpec struct {
	Get struct {
		Param int `+"`validate:\"required\"`"+`
		Res struct {
			Status200 struct {
				Body string `+"`json:\"body\" validate:\"required\"`"+`
			}
		}
	}
}
`)
	writeFile(t, api, "users/userid/route.go", `package userid

import "context"

var Route = DefineRoute(RouteHandlers{
	Get: func(ctx context.Context, req GetRequest) (GetResponse, error) {
		return GetStatus200{Body: "user"}, nil
	},
})
`)
	writeFile(t, api, "blog/slug/frourio.go", `package slug

type FrourioSpec struct {
	Get struct {
		Param []string `+"`validate:\"required\"`"+`
		Res struct {
			Status200 struct {
				Body string `+"`json:\"body\" validate:\"required\"`"+`
			}
		}
	}
}
`)
	writeFile(t, api, "files/path/frourio.go", `package path

type FrourioSpec struct {
	Get struct {
		Param *[]string
		Res struct {
			Status200 struct {
				Body string `+"`json:\"body\" validate:\"required\"`"+`
			}
		}
	}
}
`)
	writeFile(t, api, "products/sale/frourio.go", `package sale

const FrourioPath = "セール品"

type FrourioSpec struct {
	Get struct {
		Res struct {
			Status200 struct {
				Body string `+"`json:\"body\" validate:\"required\"`"+`
			}
		}
	}
}
`)
	writeFile(t, api, "mw/frourio.go", `package mw

type FrourioSpec struct {
	Middleware struct {
		All struct {
			Context struct {
				TraceID string `+"`json:\"traceId\" validate:\"required\"`"+`
			}
		}
		Get struct {
			Context struct {
				Role string `+"`json:\"role\" validate:\"required\"`"+`
			}
		}
	}
	Get struct {
		Res struct {
			Status200 struct {
				Body string `+"`json:\"body\" validate:\"required\"`"+`
			}
		}
	}
}
`)
	writeFile(t, api, "mw/child/frourio.go", `package child

type FrourioSpec struct {
	Get struct {
		Res struct {
			Status200 struct {
				Body string `+"`json:\"body\" validate:\"required\"`"+`
			}
		}
	}
}
`)

	openAPI := filepath.Join(dir, "openapi.json")
	if err := Generate(Options{APIDir: api, OpenAPIPath: openAPI}); err != nil {
		t.Fatal(err)
	}

	assertFileContains(t, filepath.Join(api, "frourio_server.go"), "mux.Handle(\"GET /api/users/{userid}\"")
	assertFileContains(t, filepath.Join(api, "frourio_server.go"), "mux.Handle(\"GET /api/files\"")
	assertFileContains(t, filepath.Join(api, "frourio_server.go"), "mux.Handle(\"GET /api/products/セール品\"")
	assertFileContains(t, filepath.Join(api, "frourio_server.go"), "values := r.PostForm")
	assertFileContains(t, filepath.Join(api, "frourio_server.go"), "values := r.MultipartForm.Value")
	assertFileContains(t, filepath.Join(api, "mw/child/frourio_relay.go"), "TraceID string")
	assertFileContains(t, filepath.Join(api, "mw/frourio_relay.go"), "type GetMiddlewareContext struct")

	data, err := os.ReadFile(openAPI)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	paths := doc["paths"].(map[string]any)
	for _, path := range []string{"/api", "/api/users/{userid}", "/api/blog/{slug}", "/api/files", "/api/files/{path}", "/api/products/セール品"} {
		if _, ok := paths[path]; !ok {
			t.Fatalf("OpenAPI path %s not found in %#v", path, paths)
		}
	}
	apiPost := paths["/api"].(map[string]any)["post"].(map[string]any)
	requestBody := apiPost["requestBody"].(map[string]any)
	content := requestBody["content"].(map[string]any)
	if _, ok := content["application/x-www-form-urlencoded"]; !ok {
		t.Fatalf("urlencoded content type not found in %#v", content)
	}
	apiPut := paths["/api"].(map[string]any)["put"].(map[string]any)
	putRequestBody := apiPut["requestBody"].(map[string]any)
	putContent := putRequestBody["content"].(map[string]any)
	if _, ok := putContent["multipart/form-data"]; !ok {
		t.Fatalf("formData content type not found in %#v", putContent)
	}
}

func TestGenerateOpenAPIOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.26\n")
	api := filepath.Join(dir, "api")
	writeFile(t, api, "frourio.go", `package api

type FrourioSpec struct {
	Get struct {
		Res struct {
			Status204 struct{}
		}
	}
}
`)

	out := filepath.Join(dir, "doc", "openapi.json")
	if err := Generate(Options{APIDir: api, OpenAPIPath: out, OnlyOpenAPI: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(api, "frourio_relay.go")); !os.IsNotExist(err) {
		t.Fatalf("frourio_relay.go should not be generated in OpenAPI-only mode, err=%v", err)
	}
	assertFileContains(t, out, `"204"`)

	dir = t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/default\n\ngo 1.26\n")
	api = filepath.Join(dir, "api")
	writeFile(t, api, "frourio.go", `package api

type FrourioSpec struct {
	Get struct {
		Res struct {
			Status204 struct{}
		}
	}
}
`)
	if err := Generate(Options{APIDir: api}); err != nil {
		t.Fatal(err)
	}
	assertFileContains(t, filepath.Join(api, "openapi.json"), `"204"`)
}

func TestGenerateErrors(t *testing.T) {
	t.Run("missing api dir", func(t *testing.T) {
		if err := Generate(Options{}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("no frourio files", func(t *testing.T) {
		dir := t.TempDir()
		if err := Generate(Options{APIDir: dir}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("unsupported directory syntax", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.26\n")
		writeFile(t, filepath.Join(dir, "api/~id"), "frourio.go", `package id
type FrourioSpec struct{}
`)
		if err := Generate(Options{APIDir: filepath.Join(dir, "api")}); err == nil || !strings.Contains(err.Error(), "~ path parameter") {
			t.Fatalf("expected ~ path parameter error, got %v", err)
		}
	})

	t.Run("method middleware requires method", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.26\n")
		writeFile(t, filepath.Join(dir, "api"), "frourio.go", `package api
type FrourioSpec struct {
	Middleware struct {
		Post bool
	}
	Get struct {
		Res struct { Status200 struct{} }
	}
}
`)
		if err := Generate(Options{APIDir: filepath.Join(dir, "api")}); err == nil || !strings.Contains(err.Error(), "Middleware.Post") {
			t.Fatalf("expected middleware method error, got %v", err)
		}
	})

	t.Run("unsupported bracket and group directories", func(t *testing.T) {
		for _, rel := range []string{"[id]", "(group)"} {
			dir := t.TempDir()
			writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.26\n")
			writeFile(t, filepath.Join(dir, "api", rel), "frourio.go", `package api
type FrourioSpec struct{}
`)
			if err := Generate(Options{APIDir: filepath.Join(dir, "api")}); err == nil {
				t.Fatalf("expected directory error for %s", rel)
			}
		}
	})

	t.Run("invalid request parts", func(t *testing.T) {
		cases := []string{
			`type FrourioSpec struct { Get struct { Query string } }`,
			`type FrourioSpec struct { Get struct { Body string } }`,
			`type FrourioSpec struct { Get struct { Header string } }`,
			`type FrourioSpec struct { Get struct { Res string } }`,
			`type FrourioSpec struct { Get struct { Res struct { Status200 string } } }`,
			`type FrourioSpec struct { Middleware string }`,
			`type FrourioSpec struct { Middleware struct { All int } }`,
			`type FrourioSpec struct { Middleware struct { All struct { Context string } } }`,
		}
		for _, spec := range cases {
			dir := t.TempDir()
			writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.26\n")
			writeFile(t, filepath.Join(dir, "api"), "frourio.go", "package api\n"+spec+"\n")
			if err := Generate(Options{APIDir: filepath.Join(dir, "api")}); err == nil {
				t.Fatalf("expected error for %s", spec)
			}
		}
	})

	t.Run("param mismatch and root param", func(t *testing.T) {
		cases := []string{
			`type FrourioSpec struct { Get struct { Param int }; Post struct{} }`,
			`type FrourioSpec struct { Get struct { Param int } }`,
		}
		for i, spec := range cases {
			dir := t.TempDir()
			writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.26\n")
			api := filepath.Join(dir, "api")
			if i == 0 {
				api = filepath.Join(api, "id")
			}
			writeFile(t, api, "frourio.go", "package api\n"+spec+"\n")
			if err := Generate(Options{APIDir: filepath.Join(dir, "api")}); err == nil {
				t.Fatalf("expected param error for %s", spec)
			}
		}
	})

	t.Run("invalid frourio spec shape", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.26\n")
		writeFile(t, filepath.Join(dir, "api"), "frourio.go", `package api
type FrourioSpec string
`)
		if err := Generate(Options{APIDir: filepath.Join(dir, "api")}); err == nil || !strings.Contains(err.Error(), "FrourioSpec must be a struct") {
			t.Fatalf("expected FrourioSpec shape error, got %v", err)
		}
	})

	t.Run("frourio spec not found", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.26\n")
		writeFile(t, filepath.Join(dir, "api"), "frourio.go", `package api
type OtherSpec struct{}
`)
		if err := Generate(Options{APIDir: filepath.Join(dir, "api")}); err == nil || !strings.Contains(err.Error(), "FrourioSpec not found") {
			t.Fatalf("expected FrourioSpec not found error, got %v", err)
		}
	})

	t.Run("invalid method shape", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.26\n")
		writeFile(t, filepath.Join(dir, "api"), "frourio.go", `package api
type FrourioSpec struct { Get string }
`)
		if err := Generate(Options{APIDir: filepath.Join(dir, "api")}); err == nil || !strings.Contains(err.Error(), "must be a struct") {
			t.Fatalf("expected method shape error, got %v", err)
		}
	})

	t.Run("invalid body format markers", func(t *testing.T) {
		cases := []string{
			`type FrourioSpec struct { Post struct { URLEncoded string; Res struct { Status204 struct{} } } }`,
			`type FrourioSpec struct { Post struct { FormData string; Res struct { Status204 struct{} } } }`,
			`type FrourioSpec struct { Post struct { URLEncoded bool; FormData bool; Res struct { Status204 struct{} } } }`,
		}
		for _, spec := range cases {
			dir := t.TempDir()
			writeFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.26\n")
			writeFile(t, filepath.Join(dir, "api"), "frourio.go", "package api\n"+spec+"\n")
			if err := Generate(Options{APIDir: filepath.Join(dir, "api")}); err == nil {
				t.Fatalf("expected format error for %s", spec)
			}
		}
	})

	t.Run("missing module", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "api"), "frourio.go", `package api
type FrourioSpec struct{}
`)
		if err := Generate(Options{APIDir: filepath.Join(dir, "api")}); err == nil || !strings.Contains(err.Error(), "go.mod not found") {
			t.Fatalf("expected go.mod error, got %v", err)
		}
	})
}

func TestHelpers(t *testing.T) {
	if got := operationID("GET", "/api/users/{userid}"); got != "getApiUsersByUserid" {
		t.Fatalf("operationID = %s", got)
	}
	if got := openAPIPath("/api/blog/{slug...}"); got != "/api/blog/{slug}" {
		t.Fatalf("openAPIPath = %s", got)
	}
	if path, ok := optionalCatchAllPath(MethodSpec{URLPath: "/api/files/{path...}", Param: &FieldSpec{Pointer: true, Slice: true}}); !ok || path != "/api/files" {
		t.Fatalf("optionalCatchAllPath = %s, %v", path, ok)
	}
	if got := routeAlias("secure/admin/users"); got != "secureAdminUsers" {
		t.Fatalf("routeAlias = %s", got)
	}
	if got := routeAlias(""); got != "route" {
		t.Fatalf("routeAlias empty = %s", got)
	}
	if got := exportName("body"); got != "Body" {
		t.Fatalf("exportName body = %s", got)
	}
	if got := exportName(""); got != "" {
		t.Fatalf("exportName empty = %s", got)
	}
	if got := lowerName(""); got != "" {
		t.Fatalf("lowerName empty = %s", got)
	}
	if !isAncestorRel("", "api") || !isAncestorRel("api", "api/users") || isAncestorRel("api/users", "api") {
		t.Fatal("isAncestorRel returned unexpected result")
	}
}

func TestSchemaAndDecodeHelpers(t *testing.T) {
	fields := []FieldSpec{
		{Name: "Name", SourceName: "Name", Type: "string", JSONName: "name", ValidateTag: "required"},
		{Name: "Age", SourceName: "Age", Type: "int", JSONName: "age"},
		{Name: "Enabled", SourceName: "Enabled", Type: "bool", JSONName: "enabled"},
		{Name: "Scores", SourceName: "Scores", Type: "[]int", JSONName: "scores", Slice: true},
		{Name: "When", SourceName: "When", Type: "time.Time", JSONName: "when"},
		{Name: "Rate", SourceName: "Rate", Type: "*float64", JSONName: "rate", Pointer: true},
	}
	schema := schemaForStruct(&StructSpec{Name: "Body", Fields: fields})
	if schema["type"] != "object" {
		t.Fatalf("schemaForStruct type = %#v", schema)
	}
	for _, field := range fields {
		if got := schemaForField(field); got["type"] == "" {
			t.Fatalf("schemaForField(%s) missing type", field.Type)
		}
	}

	var b strings.Builder
	writeQueryDecode(&b, FieldSpec{Name: "Names", SourceName: "Names", Type: "[]string", JSONName: "name", Slice: true})
	writeQueryDecode(&b, FieldSpec{Name: "Ids", SourceName: "Ids", Type: "[]int", JSONName: "id", Slice: true})
	writeQueryDecode(&b, FieldSpec{Name: "Title", SourceName: "Title", Type: "string", JSONName: "title"})
	writeQueryDecode(&b, FieldSpec{Name: "Count", SourceName: "Count", Type: "int", JSONName: "count"})
	writeQueryDecode(&b, FieldSpec{Name: "OptionalCount", SourceName: "OptionalCount", Type: "*int", JSONName: "optionalCount", Pointer: true})
	writeQueryDecode(&b, FieldSpec{Name: "Enabled", SourceName: "Enabled", Type: "*bool", JSONName: "enabled", Pointer: true})
	writeQueryDecode(&b, FieldSpec{Name: "Bad", SourceName: "Bad", Type: "custom.Type", JSONName: "bad"})
	text := b.String()
	for _, want := range []string{"values[\"name\"]", "decodeInt", "unsupported values type"} {
		if !strings.Contains(text, want) {
			t.Fatalf("writeQueryDecode missing %q in %s", want, text)
		}
	}

	b.Reset()
	writeParamDecode(&b, RouteSpec{RelDir: "id"}, FieldSpec{Type: "*string", Pointer: true})
	writeParamDecode(&b, RouteSpec{RelDir: "id"}, FieldSpec{Type: "int"})
	writeParamDecode(&b, RouteSpec{RelDir: "slug"}, FieldSpec{Type: "[]int", Slice: true})
	writeParamDecode(&b, RouteSpec{RelDir: "id"}, FieldSpec{Type: "float64"})
	if !strings.Contains(b.String(), "unsupported param type") {
		t.Fatalf("writeParamDecode missing unsupported branch: %s", b.String())
	}

	for _, status := range []int{200, 201, 204, 400, 404, 418} {
		if httpStatusDescription(status) == "" {
			t.Fatalf("empty status description for %d", status)
		}
	}
	for _, method := range []string{"get", "post", "put", "patch", "delete", "head", "options", "trace"} {
		httpMethodName(method)
	}
	if _, ok := parseStatusName("Status999"); ok {
		t.Fatal("unexpected valid status")
	}
	if err := writeGoFile(filepath.Join(t.TempDir(), "bad.go"), "package {"); err == nil {
		t.Fatal("expected writeGoFile format error")
	}
	if got := firstMethodPath(RouteSpec{URLPath: "/api"}); got != "/api" {
		t.Fatalf("firstMethodPath fallback = %s", got)
	}
	if got := rootPackage([]RouteSpec{{RelDir: "users", PackageName: "users"}}); got != "users" {
		t.Fatalf("rootPackage fallback = %s", got)
	}
	if got := pathParamName(""); got != "param" {
		t.Fatalf("pathParamName root = %s", got)
	}
	if _, ok := optionalCatchAllPath(MethodSpec{URLPath: "/api/files/{path}", Param: &FieldSpec{Pointer: true, Slice: true}}); ok {
		t.Fatal("unexpected optional catch-all")
	}
	if got := routePath("admin//users", map[string]string{"admin/users": "members"}, nil); got != "/api/admin/users" {
		t.Fatalf("routePath empty part = %s", got)
	}
	if _, err := parseStruct("Bad", &ast.Ident{Name: "string"}); err == nil {
		t.Fatal("expected parseStruct error")
	}
	if got := exprString(&ast.SelectorExpr{X: &ast.Ident{Name: "time"}, Sel: &ast.Ident{Name: "Time"}}); got != "time.Time" {
		t.Fatalf("exprString selector = %s", got)
	}
	if got := exprString(&ast.ArrayType{Elt: &ast.Ident{Name: "string"}}); got != "[]string" {
		t.Fatalf("exprString array = %s", got)
	}
	if got := exprString(&ast.StructType{}); got != "any" {
		t.Fatalf("exprString default = %s", got)
	}
	if tag := goTag(&FieldSpec{}, ""); tag != "" {
		t.Fatalf("empty goTag = %s", tag)
	}
	if tag := goTag(&FieldSpec{ValidateTag: "required"}, "value"); tag != "`json:\"value\" validate:\"required\"`" {
		t.Fatalf("goTag fallback = %s", tag)
	}
}

func writeFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s does not contain %q\n%s", path, want, data)
	}
}
