package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const traceID = "trace-test-001"

func decodeJSON(t *testing.T, body string, dst any) {
	t.Helper()
	if err := json.Unmarshal([]byte(body), dst); err != nil {
		t.Fatalf("decode json failed: %v\nbody: %s", err, body)
	}
}

func TestRoot(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	if strings.TrimSpace(res.Body.String()) != `root` {
		t.Fatalf("unexpected body: %s", res.Body.String())
	}
}

func TestMwGet_NoHeaders(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/mw", nil)
	res := httptest.NewRecorder()
	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	var body struct {
		UserID  *string `json:"userId"`
		TraceID string  `json:"traceId"`
	}
	decodeJSON(t, res.Body.String(), &body)
	if body.UserID != nil {
		t.Fatalf("expected userId nil, got %q", *body.UserID)
	}
	if body.TraceID == "" {
		t.Fatalf("expected traceId to be generated, got empty")
	}
}

func TestMwGet_UserAuthorization(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/mw", nil)
	req.Header.Set("Authorization", "Bearer user-123")
	req.Header.Set("X-Trace-Id", traceID)
	res := httptest.NewRecorder()
	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	var body struct {
		UserID  *string `json:"userId"`
		TraceID string  `json:"traceId"`
	}
	decodeJSON(t, res.Body.String(), &body)
	if body.UserID == nil || *body.UserID != "user-123" {
		t.Fatalf("expected userId=user-123, got %v", body.UserID)
	}
	if body.TraceID != traceID {
		t.Fatalf("expected traceId=%s, got %s", traceID, body.TraceID)
	}
}

func TestMwGet_AdminAuthorization(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/mw", nil)
	req.Header.Set("Authorization", "Bearer user-admin")
	req.Header.Set("X-Trace-Id", traceID)
	res := httptest.NewRecorder()
	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	var body struct {
		UserID  *string `json:"userId"`
		TraceID string  `json:"traceId"`
	}
	decodeJSON(t, res.Body.String(), &body)
	if body.UserID == nil || *body.UserID != "user-admin" {
		t.Fatalf("expected userId=user-admin, got %v", body.UserID)
	}
	if body.TraceID != traceID {
		t.Fatalf("expected traceId=%s, got %s", traceID, body.TraceID)
	}
}

type adminContextResp struct {
	UserID      *string  `json:"userId,omitempty"`
	TraceID     string   `json:"traceId"`
	Permissions []string `json:"permissions"`
	IsAdmin     bool     `json:"isAdmin"`
}

func TestAdminGet_AdminUser(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/mw/admin", nil)
	req.Header.Set("Authorization", "Bearer user-admin")
	req.Header.Set("X-Trace-Id", traceID)
	res := httptest.NewRecorder()
	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	var body adminContextResp
	decodeJSON(t, res.Body.String(), &body)
	if body.UserID == nil || *body.UserID != "user-admin" {
		t.Fatalf("expected userId=user-admin, got %v", body.UserID)
	}
	if body.TraceID != traceID {
		t.Fatalf("expected traceId=%s, got %s", traceID, body.TraceID)
	}
	if !body.IsAdmin {
		t.Fatalf("expected isAdmin=true")
	}
	if got, want := body.Permissions, []string{"read", "write", "delete"}; !equalStrings(got, want) {
		t.Fatalf("expected permissions=%v, got %v", want, got)
	}
}

func TestAdminGet_NormalUser(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/mw/admin", nil)
	req.Header.Set("Authorization", "Bearer user-regular")
	req.Header.Set("X-Trace-Id", traceID)
	res := httptest.NewRecorder()
	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	var body adminContextResp
	decodeJSON(t, res.Body.String(), &body)
	if body.IsAdmin {
		t.Fatalf("expected isAdmin=false")
	}
	if got, want := body.Permissions, []string{"read"}; !equalStrings(got, want) {
		t.Fatalf("expected permissions=%v, got %v", want, got)
	}
	if body.UserID == nil || *body.UserID != "user-regular" {
		t.Fatalf("expected userId=user-regular, got %v", body.UserID)
	}
}

func TestAdminPost_AdminUser(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/mw/admin", strings.NewReader(`{"data":"admin data"}`))
	req.Header.Set("Authorization", "Bearer user-admin")
	req.Header.Set("X-Trace-Id", traceID)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", res.Code, res.Body.String())
	}
	var body struct {
		Received string           `json:"received"`
		Context  adminContextResp `json:"context"`
	}
	decodeJSON(t, res.Body.String(), &body)
	if body.Received != "admin data" {
		t.Fatalf("expected received=admin data, got %s", body.Received)
	}
	if body.Context.UserID == nil || *body.Context.UserID != "user-admin" {
		t.Fatalf("expected context.userId=user-admin, got %v", body.Context.UserID)
	}
	if !body.Context.IsAdmin {
		t.Fatalf("expected context.isAdmin=true")
	}
}

func TestAdminPost_NormalUser_Forbidden(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/mw/admin", strings.NewReader(`{"data":"user data"}`))
	req.Header.Set("Authorization", "Bearer user-regular")
	req.Header.Set("X-Trace-Id", traceID)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", res.Code, res.Body.String())
	}
	var body struct {
		Message string `json:"message"`
	}
	decodeJSON(t, res.Body.String(), &body)
	if body.Message != "Forbidden: Admin access required" {
		t.Fatalf("expected forbidden message, got %s", body.Message)
	}
}

func TestAdminPost_NoAuth_Forbidden(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/mw/admin", strings.NewReader(`{"data":"no auth data"}`))
	req.Header.Set("X-Trace-Id", traceID)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", res.Code, res.Body.String())
	}
	var body struct {
		Message string `json:"message"`
	}
	decodeJSON(t, res.Body.String(), &body)
	if body.Message != "Forbidden: Admin access required" {
		t.Fatalf("expected forbidden message, got %s", body.Message)
	}
}

func TestAdminPost_InvalidBody(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/mw/admin", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer user-admin")
	req.Header.Set("X-Trace-Id", traceID)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", res.Code, res.Body.String())
	}
}

func TestAdminUsersGet_AdminUser(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/mw/admin/users?role=admin", nil)
	req.Header.Set("Authorization", "Bearer user-admin")
	req.Header.Set("X-Trace-Id", traceID)
	res := httptest.NewRecorder()
	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	var body struct {
		Users   []string         `json:"users"`
		Context adminContextResp `json:"context"`
	}
	decodeJSON(t, res.Body.String(), &body)
	if !body.Context.IsAdmin {
		t.Fatalf("expected admin context")
	}
	if got, want := body.Users, []string{"admin1"}; !equalStrings(got, want) {
		t.Fatalf("expected users=%v, got %v", want, got)
	}
}

func TestAdminUsersGet_NormalUser(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/mw/admin/users", nil)
	req.Header.Set("Authorization", "Bearer user-regular")
	req.Header.Set("X-Trace-Id", traceID)
	res := httptest.NewRecorder()
	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	var body struct {
		Users   []string         `json:"users"`
		Context adminContextResp `json:"context"`
	}
	decodeJSON(t, res.Body.String(), &body)
	if body.Context.IsAdmin {
		t.Fatalf("expected non-admin context")
	}
	if len(body.Users) != 0 {
		t.Fatalf("expected empty users for non-admin, got %v", body.Users)
	}
}

func TestPublicGet_NoHeaders(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/public", nil)
	res := httptest.NewRecorder()
	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	var body struct {
		Message string `json:"message"`
	}
	decodeJSON(t, res.Body.String(), &body)
	if body.Message != "This is a public endpoint." {
		t.Fatalf("expected public message, got %s", body.Message)
	}
}

func TestPublicGet_WithAuthorizationIgnored(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/public", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	res := httptest.NewRecorder()
	Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	var body struct {
		Message string `json:"message"`
	}
	decodeJSON(t, res.Body.String(), &body)
	if body.Message != "This is a public endpoint." {
		t.Fatalf("expected public message, got %s", body.Message)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
