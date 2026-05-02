package api

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerGet(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api?search=hello&limit=10&RawName=raw&active=true&score=1.5&score=2.5", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `ok` {
		t.Fatalf("expected text body, got %s", res.Body.String())
	}
}

func TestHandlerGetInvalidQueryBool(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api?active=maybe", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d: %s", res.Code, res.Body.String())
	}
}

func TestHandlerGetInvalidQuery(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api?limit=bad", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d: %s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "Unprocessable Entity") {
		t.Fatalf("expected validation error body, got %s", res.Body.String())
	}
}

func TestHandlerUsersGet(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/users?limit=1", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `["alice"]` {
		t.Fatalf("expected filtered users body, got %s", res.Body.String())
	}
}

func TestHandlerUsersPost(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/users", strings.NewReader(`{"name":"alice","age":20}`))
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `alice:20` {
		t.Fatalf("expected created user body, got %s", res.Body.String())
	}
}

func TestHandlerUsersPostInvalidBody(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/users", strings.NewReader(`{"age":20}`))
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d: %s", res.Code, res.Body.String())
	}
}

func TestHandlerFormsPostURLEncoded(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/forms", strings.NewReader("name=alice&Alias=ally&age=20&active=true&score=1.5&score=2.5"))
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `alice:ally:20:true:2` {
		t.Fatalf("expected urlencoded body decode result, got %s", res.Body.String())
	}
}

func TestHandlerFormsGetText(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/forms", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if res.Header().Get("content-type") != "text/custom" {
		t.Fatalf("expected custom content-type, got %s", res.Header().Get("content-type"))
	}
	if res.Body.String() != "plain" {
		t.Fatalf("expected text body, got %s", res.Body.String())
	}
}

func TestHandlerFormsPatchBytes(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/forms", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if res.Header().Get("content-type") != "application/octet-stream" {
		t.Fatalf("expected octet-stream content-type, got %s", res.Header().Get("content-type"))
	}
	if !bytes.Equal(res.Body.Bytes(), []byte{1, 2, 3}) {
		t.Fatalf("expected binary body, got %v", res.Body.Bytes())
	}
}

func TestHandlerFormsDeleteMultipartResponse(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/forms", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if !strings.HasPrefix(res.Header().Get("content-type"), "multipart/form-data; boundary=") {
		t.Fatalf("expected multipart content-type, got %s", res.Header().Get("content-type"))
	}
	if !strings.Contains(res.Body.String(), `name="name"`) || !strings.Contains(res.Body.String(), "alice") {
		t.Fatalf("expected multipart name field, got %s", res.Body.String())
	}
	if !strings.Contains(res.Body.String(), `name="Count"`) || !strings.Contains(res.Body.String(), "2") {
		t.Fatalf("expected multipart Count field, got %s", res.Body.String())
	}
}

func TestHandlerFormsPostInvalidURLEncoded(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/forms", strings.NewReader("name=alice&age=bad&active=true&score=1.5"))
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d: %s", res.Code, res.Body.String())
	}
}

func TestHandlerRawStreamGet(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/raw", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if res.Header().Get("content-type") != "text/plain" {
		t.Fatalf("expected raw content-type, got %s", res.Header().Get("content-type"))
	}
	if res.Body.String() != "chunk-1\nchunk-2\n" {
		t.Fatalf("expected raw streamed body, got %s", res.Body.String())
	}
}

func TestHandlerFormsPutMultipart(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("title", "report"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("count", "3"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/forms", &body)
	req.Header.Set("content-type", writer.FormDataContentType())
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `report:3` {
		t.Fatalf("expected multipart body decode result, got %s", res.Body.String())
	}
}

func TestHandlerUserGet(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/users/123", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `user:123` {
		t.Fatalf("expected user body, got %s", res.Body.String())
	}
}

func TestHandlerProductsSaleGet(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/products/セール品", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `sale` {
		t.Fatalf("expected sale body, got %s", res.Body.String())
	}
}

func TestHandlerBlogCatchAllGet(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/blog/a/b/c", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `a/b/c` {
		t.Fatalf("expected catch-all body, got %s", res.Body.String())
	}
}

func TestHandlerFilesOptionalCatchAllRootGet(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/files", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `root` {
		t.Fatalf("expected optional catch-all root body, got %s", res.Body.String())
	}
}

func TestHandlerFilesOptionalCatchAllPathGet(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/files/a/b", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `a/b` {
		t.Fatalf("expected optional catch-all path body, got %s", res.Body.String())
	}
}

func TestHandlerMiddlewareGet(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/mw", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `admin` {
		t.Fatalf("expected middleware context body, got %s", res.Body.String())
	}
}

func TestHandlerMiddlewareAllContextGet(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `trace-123` {
		t.Fatalf("expected middleware all context body, got %s", res.Body.String())
	}
}

func TestHandlerInheritedMiddlewareContextGet(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/nest/child", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `nested-trace` {
		t.Fatalf("expected inherited middleware context body, got %s", res.Body.String())
	}
}

func TestHandlerSecureRootMiddlewareContextGet(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/secure", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `user-admin:trace-root` {
		t.Fatalf("expected root middleware context body, got %s", res.Body.String())
	}
}

func TestHandlerSecureAdminNestedMiddlewareGet(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/secure/admin", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	expected := `user-admin:trace-root:true:read,write,delete`
	if strings.TrimSpace(res.Body.String()) != expected {
		t.Fatalf("expected nested middleware context body %s, got %s", expected, res.Body.String())
	}
}

func TestHandlerSecureAdminPostMiddleware(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/secure/admin", strings.NewReader(`{"data":"admin data"}`))
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", res.Code, res.Body.String())
	}
	expected := `admin data:user-admin:true:read,write,delete`
	if strings.TrimSpace(res.Body.String()) != expected {
		t.Fatalf("expected admin post body %s, got %s", expected, res.Body.String())
	}
}

func TestHandlerSecureAdminUsersMiddlewareInheritanceGet(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/secure/admin/users?role=admin", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	expected := `user-admin:trace-root:true:read,write,delete:admin`
	if strings.TrimSpace(res.Body.String()) != expected {
		t.Fatalf("expected inherited middleware context body %s, got %s", expected, res.Body.String())
	}
}

func TestHandlerSecurePublicNoMiddlewareGet(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/public", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `public` {
		t.Fatalf("expected public body, got %s", res.Body.String())
	}
}

func TestHandlerUserGetInvalidParam(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/users/bad", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d: %s", res.Code, res.Body.String())
	}
}

func TestHandlerGetInvalidQueryCases(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"limit not number", "/api?limit=bad"},
		{"active not bool", "/api?active=maybe"},
		{"score not float", "/api?score=abc"},
		{"score mixed", "/api?score=1.0&score=bad"},
		{"limit float", "/api?limit=1.5"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, c.url, nil)
			res := httptest.NewRecorder()
			Handler().ServeHTTP(res, req)
			if res.Code != http.StatusUnprocessableEntity {
				t.Fatalf("expected 422, got %d: %s", res.Code, res.Body.String())
			}
		})
	}
}

func TestHandlerGetValidQueryCases(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"all fields", "/api?search=hello&limit=10&RawName=raw&active=true&score=1.5&score=2.5"},
		{"only optional empty", "/api"},
		{"empty score", "/api?active=false"},
		{"single score", "/api?score=0.5"},
		{"limit zero", "/api?limit=0"},
		{"active false", "/api?active=false"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, c.url, nil)
			res := httptest.NewRecorder()
			Handler().ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
			}
		})
	}
}

func TestHandlerUsersPostInvalidBodyCases(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"missing name", `{"age":20}`},
		{"name not string", `{"name":1}`},
		{"age not number", `{"name":"alice","age":"twenty"}`},
		{"empty body", ``},
		{"invalid json", `{"name":}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/users", strings.NewReader(c.body))
			res := httptest.NewRecorder()
			Handler().ServeHTTP(res, req)
			if res.Code == http.StatusCreated {
				t.Fatalf("expected error, got 201: %s", res.Body.String())
			}
		})
	}
}

func TestHandlerFormsPostURLEncodedInvalidCases(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"missing name", "age=20&active=true&score=1.0"},
		{"missing scores", "name=alice&age=20&active=true"},
		{"age zero violates gte=1", "name=alice&age=0&active=true&score=1.0"},
		{"age not number", "name=alice&age=bad&active=true&score=1.0"},
		{"score not number", "name=alice&age=20&active=true&score=bad"},
		{"active not bool", "name=alice&age=20&active=maybe&score=1.0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/forms", strings.NewReader(c.body))
			req.Header.Set("content-type", "application/x-www-form-urlencoded")
			res := httptest.NewRecorder()
			Handler().ServeHTTP(res, req)
			if res.Code != http.StatusUnprocessableEntity {
				t.Fatalf("expected 422, got %d: %s", res.Code, res.Body.String())
			}
		})
	}
}

func TestHandlerFormsPutMultipartInvalidCases(t *testing.T) {
	cases := []struct {
		fields map[string]string
		name   string
	}{
		{map[string]string{"count": "3"}, "missing title"},
		{map[string]string{"title": "report"}, "missing count"},
		{map[string]string{"title": "report", "count": "0"}, "count zero violates gte=1"},
		{map[string]string{"title": "report", "count": "abc"}, "count not number"},
		{map[string]string{"title": "", "count": "3"}, "empty title"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			for k, v := range c.fields {
				if err := writer.WriteField(k, v); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/forms", &body)
			req.Header.Set("content-type", writer.FormDataContentType())
			res := httptest.NewRecorder()
			Handler().ServeHTTP(res, req)
			if res.Code != http.StatusOK && res.Code != http.StatusUnprocessableEntity {
				t.Fatalf("expected 200 or 422, got %d: %s", res.Code, res.Body.String())
			}
			if res.Code == http.StatusOK {
				t.Fatalf("expected validation failure (422), got 200: %s", res.Body.String())
			}
		})
	}
}

func TestHandlerUserGetInvalidParamCases(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"non-numeric", "/api/users/bad"},
		{"float", "/api/users/1.5"},
		{"empty after slash", "/api/users/"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, c.url, nil)
			res := httptest.NewRecorder()
			Handler().ServeHTTP(res, req)
			if res.Code == http.StatusOK {
				t.Fatalf("expected error, got 200: %s", res.Body.String())
			}
		})
	}
}

func TestHandlerForms_GetTextHeaderEcho(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/forms", nil)
	res := httptest.NewRecorder()
	Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if got := res.Header().Get("content-type"); got != "text/custom" {
		t.Fatalf("expected content-type=text/custom, got %s", got)
	}
}

func TestHandlerSecureAdminPostMiddleware_InvalidBody(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/secure/admin", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	Handler().ServeHTTP(res, req)
	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", res.Code, res.Body.String())
	}
}
