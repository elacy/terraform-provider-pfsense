package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func envelope(data any) map[string]any {
	return map[string]any{
		"code":        200,
		"status":      "ok",
		"response_id": "SUCCESS",
		"message":     "",
		"data":        data,
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode: %v", err)
	}
}

func TestClientBasicAuthHeader(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Basic YWRtaW46cGZzZW5zZQ==" { // admin:pfsense
			t.Errorf("Authorization = %q", got)
		}
		writeJSON(t, w, 200, envelope([]any{}))
	})
	c, err := New(Config{URL: srv.URL, Username: "admin", Password: "pfsense"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.List(context.Background(), "/api/v2/firewall/aliases", nil); err != nil {
		t.Fatal(err)
	}
}

func TestClientAPIKeyHeader(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-API-Key"); got != "sekret-key" {
			t.Errorf("X-API-Key = %q", got)
		}
		writeJSON(t, w, 200, envelope([]any{}))
	})
	c, err := New(Config{URL: srv.URL, APIKey: "sekret-key"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.List(context.Background(), "/api/v2/firewall/aliases", nil); err != nil {
		t.Fatal(err)
	}
}

func TestClientJWTAuthAndRefresh(t *testing.T) {
	var jwtCalls int
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v2/auth/jwt":
			jwtCalls++
			writeJSON(t, w, 200, envelope(map[string]any{"token": "test-jwt"}))
		case strings.HasPrefix(r.URL.Path, "/api/v2/"):
			if got := r.Header.Get("Authorization"); got != "Bearer test-jwt" {
				t.Errorf("Authorization = %q", got)
			}
			writeJSON(t, w, 200, envelope([]any{}))
		}
	})
	c, err := New(Config{URL: srv.URL, Username: "admin", Password: "pfsense", UseJWT: true})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	// First call issues a JWT, second uses the cache.
	for i := 0; i < 2; i++ {
		if _, err := c.List(ctx, "/api/v2/firewall/aliases", nil); err != nil {
			t.Fatal(err)
		}
	}
	if jwtCalls != 1 {
		t.Errorf("expected 1 JWT call (cached), got %d", jwtCalls)
	}
	// Simulate an expired token -> refresh path.
	c.invalidateJWT()
	if _, err := c.List(ctx, "/api/v2/firewall/aliases", nil); err != nil {
		t.Fatal(err)
	}
	if jwtCalls != 2 {
		t.Errorf("expected 2 JWT calls after invalidate, got %d", jwtCalls)
	}
}

func TestClientJWTRefreshOn401(t *testing.T) {
	var jwtCalls int
	var allow bool
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v2/auth/jwt":
			jwtCalls++
			allow = true
			writeJSON(t, w, 200, envelope(map[string]any{"token": "fresh-jwt"}))
		case !allow:
			writeJSON(t, w, 401, map[string]any{"code": 401, "status": "unauthorized", "response_id": "AUTHENTICATION_FAILED", "message": "nope"})
		default:
			if got := r.Header.Get("Authorization"); got != "Bearer fresh-jwt" {
				t.Errorf("Authorization = %q", got)
			}
			writeJSON(t, w, 200, envelope([]any{}))
		}
	})
	c, err := New(Config{URL: srv.URL, Username: "admin", Password: "pfsense", UseJWT: true})
	if err != nil {
		t.Fatal(err)
	}
	// Prime the (stale) token by issuing once, then force server to reject it.
	c.jwtMu.Lock()
	c.jwtToken = "stale-jwt"
	c.jwtAt = time.Now()
	c.jwtMu.Unlock()
	if _, err := c.List(context.Background(), "/api/v2/firewall/aliases", nil); err != nil {
		t.Fatal(err)
	}
	if jwtCalls != 1 {
		t.Errorf("expected a refresh JWT call, got %d", jwtCalls)
	}
}

func TestClientAPIError(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 422, map[string]any{
			"code": 422, "status": "unprocessable entity", "response_id": "VALIDATION_ERROR", "message": "Field `name` is required.",
		})
	})
	c, _ := New(Config{URL: srv.URL, APIKey: "k"})
	_, err := c.Create(context.Background(), "/api/v2/firewall/alias", map[string]any{})
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.ResponseID != "VALIDATION_ERROR" || apiErr.Code != 422 {
		t.Errorf("unexpected APIError: %+v", apiErr)
	}
}

func TestClientGetNotFound(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 404, map[string]any{
			"code": 404, "status": "not found", "response_id": "NOT_FOUND", "message": "Object not found",
		})
	})
	c, _ := New(Config{URL: srv.URL, APIKey: "k"})
	_, err := c.Get(context.Background(), "/api/v2/firewall/alias", Query{}.Set("id", "99"))
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestClientListDecodesArray(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("name__exact") != "webservers" {
			t.Errorf("missing query filter, got %v", r.URL.Query())
		}
		writeJSON(t, w, 200, envelope([]map[string]any{{"name": "webservers", "id": 0}}))
	})
	c, _ := New(Config{URL: srv.URL, APIKey: "k"})
	items, err := c.List(context.Background(), "/api/v2/firewall/aliases", Query{}.Set("name__exact", "webservers"))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	var obj map[string]any
	if err := json.Unmarshal(items[0], &obj); err != nil {
		t.Fatal(err)
	}
	if obj["name"] != "webservers" {
		t.Errorf("unexpected object: %v", obj)
	}
}

func TestClientGetDecodesObject(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, 200, envelope(map[string]any{"name": "webservers", "id": 3, "type": "host"}))
	})
	c, _ := New(Config{URL: srv.URL, APIKey: "k"})
	raw, err := c.Get(context.Background(), "/api/v2/firewall/alias", Query{}.Set("id", "3"))
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}
	if obj["name"] != "webservers" || obj["type"] != "host" {
		t.Errorf("unexpected object: %v", obj)
	}
}

func TestNewRejectsBadURL(t *testing.T) {
	if _, err := New(Config{URL: "ftp://nope"}); err == nil {
		t.Fatal("expected error for non-http(s) URL")
	}
}
