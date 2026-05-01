package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api?search=hello&limit=10", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `"ok"` {
		t.Fatalf("expected JSON string body, got %s", res.Body.String())
	}
}

func TestHandlerGetInvalidQuery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api?limit=bad", nil)
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
	req := httptest.NewRequest(http.MethodGet, "/api/users?limit=1", nil)
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
	req := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(`{"name":"alice","age":20}`))
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `"alice:20"` {
		t.Fatalf("expected created user body, got %s", res.Body.String())
	}
}

func TestHandlerUsersPostInvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(`{"age":20}`))
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d: %s", res.Code, res.Body.String())
	}
}

func TestHandlerUserGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/users/123", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `"user:123"` {
		t.Fatalf("expected user body, got %s", res.Body.String())
	}
}

func TestHandlerProductsSaleGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/products/セール品", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `"sale"` {
		t.Fatalf("expected sale body, got %s", res.Body.String())
	}
}

func TestHandlerBlogCatchAllGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/blog/a/b/c", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `"a/b/c"` {
		t.Fatalf("expected catch-all body, got %s", res.Body.String())
	}
}

func TestHandlerFilesOptionalCatchAllRootGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/files", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `"root"` {
		t.Fatalf("expected optional catch-all root body, got %s", res.Body.String())
	}
}

func TestHandlerFilesOptionalCatchAllPathGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/files/a/b", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `"a/b"` {
		t.Fatalf("expected optional catch-all path body, got %s", res.Body.String())
	}
}

func TestHandlerMiddlewareGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/mw", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `"admin"` {
		t.Fatalf("expected middleware context body, got %s", res.Body.String())
	}
}

func TestHandlerMiddlewareAllContextGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/auth", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `"trace-123"` {
		t.Fatalf("expected middleware all context body, got %s", res.Body.String())
	}
}

func TestHandlerInheritedMiddlewareContextGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/nest/child", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `"nested-trace"` {
		t.Fatalf("expected inherited middleware context body, got %s", res.Body.String())
	}
}

func TestHandlerSecureRootMiddlewareContextGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/secure", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `"user-admin:trace-root"` {
		t.Fatalf("expected root middleware context body, got %s", res.Body.String())
	}
}

func TestHandlerSecureAdminNestedMiddlewareGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/secure/admin", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	expected := `"user-admin:trace-root:true:read,write,delete"`
	if strings.TrimSpace(res.Body.String()) != expected {
		t.Fatalf("expected nested middleware context body %s, got %s", expected, res.Body.String())
	}
}

func TestHandlerSecureAdminPostMiddleware(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/secure/admin", strings.NewReader(`{"data":"admin data"}`))
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", res.Code, res.Body.String())
	}
	expected := `"admin data:user-admin:true:read,write,delete"`
	if strings.TrimSpace(res.Body.String()) != expected {
		t.Fatalf("expected admin post body %s, got %s", expected, res.Body.String())
	}
}

func TestHandlerSecureAdminUsersMiddlewareInheritanceGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/secure/admin/users?role=admin", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	expected := `"user-admin:trace-root:true:read,write,delete:admin"`
	if strings.TrimSpace(res.Body.String()) != expected {
		t.Fatalf("expected inherited middleware context body %s, got %s", expected, res.Body.String())
	}
}

func TestHandlerSecurePublicNoMiddlewareGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/public", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `"public"` {
		t.Fatalf("expected public body, got %s", res.Body.String())
	}
}

func TestHandlerUserGetInvalidParam(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/users/bad", nil)
	res := httptest.NewRecorder()

	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d: %s", res.Code, res.Body.String())
	}
}
