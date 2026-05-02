package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMainFunctionExitsOnError(t *testing.T) {
	oldArgs := os.Args
	oldExit := osExit
	defer func() {
		os.Args = oldArgs
		osExit = oldExit
	}()

	var code int
	os.Args = []string{"frourio-go"}
	osExit = func(c int) { code = c }
	main()
	if code != 1 {
		t.Fatalf("exit code = %d", code)
	}
}

func TestRunErrors(t *testing.T) {
	tests := [][]string{
		nil,
		{"generate"},
		{"generate", "api", "--openapi"},
		{"generate", "api", "--template"},
		{"generate", "api", "--watch"},
		{"generate", "api", "--bad"},
		{"openapi"},
		{"openapi", "api"},
		{"openapi", "api", "--output"},
		{"openapi", "api", "--output", "/tmp/x.json", "--template"},
		{"openapi", "api", "--bad"},
		{"unknown"},
	}

	for _, args := range tests {
		if err := run(args); err == nil {
			t.Fatalf("run(%v) succeeded unexpectedly", args)
		}
	}
}

func TestRunGenerateAndOpenAPI(t *testing.T) {
	dir := t.TempDir()
	writeMainTestFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.26\n")
	writeMainTestFile(t, dir, "api/frourio.go", `package api

type FrourioSpec struct {
	Get struct {
		Res struct {
			Status204 struct{}
		}
	}
}
`)

	api := filepath.Join(dir, "api")
	openAPI := filepath.Join(dir, "openapi.json")
	if err := run([]string{"generate", api, "--openapi", openAPI}); err != nil {
		t.Fatal(err)
	}
	assertMainTestFileContains(t, filepath.Join(api, "frourio_relay.go"), "type GetRequest")
	assertMainTestFileContains(t, filepath.Join(api, "frourio_server.go"), "func Handler")
	assertMainTestFileContains(t, openAPI, `"openapi"`)

	openAPIOnly := filepath.Join(dir, "openapi-only.json")
	if err := run([]string{"openapi", api, "--output", openAPIOnly}); err != nil {
		t.Fatal(err)
	}
	assertMainTestFileContains(t, openAPIOnly, `"204"`)

	// Custom --template path on both subcommands.
	tmpl := filepath.Join(dir, "custom-template.json")
	writeMainTestFile(t, dir, "custom-template.json", `{"openapi":"3.0.3","info":{"title":"custom","version":"1.0.0"},"servers":[{"url":"https://example.test"}]}`)
	openAPI2 := filepath.Join(dir, "openapi2.json")
	if err := run([]string{"generate", api, "--openapi", openAPI2, "--template", tmpl}); err != nil {
		t.Fatal(err)
	}
	assertMainTestFileContains(t, openAPI2, `"https://example.test"`)
	openAPIOnly2 := filepath.Join(dir, "openapi-only2.json")
	if err := run([]string{"openapi", api, "--output", openAPIOnly2, "--template", tmpl}); err != nil {
		t.Fatal(err)
	}
	assertMainTestFileContains(t, openAPIOnly2, `"custom"`)
}

func writeMainTestFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertMainTestFileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s does not contain %q\n%s", path, want, data)
	}
}
