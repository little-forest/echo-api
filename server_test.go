package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const allowedMethods = "OPTIONS, GET"

func serve(t *testing.T, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	newHandler().ServeHTTP(rec, req)
	assertSecurityHeaders(t, rec)
	return rec
}

func assertSecurityHeaders(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	h := rec.Header()
	if got := h.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
	if got := h.Get("X-Frame-Options"); got != "SAMEORIGIN" {
		t.Errorf("X-Frame-Options = %q, want %q", got, "SAMEORIGIN")
	}
	if got := h.Get("X-Xss-Protection"); got != "1; mode=block" {
		t.Errorf("X-Xss-Protection = %q, want %q", got, "1; mode=block")
	}
}

func decodeDump(t *testing.T, rec *httptest.ResponseRecorder) DumpResult {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}
	body := rec.Body.String()
	if !strings.HasSuffix(body, "}\n") {
		t.Errorf("body does not end with newline: %q", body)
	}
	var r DumpResult
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		t.Fatalf("decode body: %v (body=%q)", err, body)
	}
	if r.Time == "" || r.RemoteAddr == "" || r.Headers == nil {
		t.Errorf("missing fields in %+v", r)
	}
	return r
}

func TestGet(t *testing.T) {
	t.Setenv("X_APP_NAME", "")

	req := httptest.NewRequest(http.MethodGet, "/foo/bar?x=1", nil)
	req.Header.Set("X-Test", "hello")
	r := decodeDump(t, serve(t, req))

	if r.Path != "/foo/bar?x=1" {
		t.Errorf("path = %q, want %q", r.Path, "/foo/bar?x=1")
	}
	if got := r.Headers["X-Test"]; got != "hello" {
		t.Errorf("headers[X-Test] = %q, want %q", got, "hello")
	}
	if _, ok := r.Headers["X-Application-Name"]; ok {
		t.Errorf("headers[X-Application-Name] exists without X_APP_NAME")
	}
}

func TestGetMultiHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add("X-Multi", "a")
	req.Header.Add("X-Multi", "b")
	r := decodeDump(t, serve(t, req))

	if got := r.Headers["X-Multi"]; got != "b" {
		t.Errorf("headers[X-Multi] = %q, want %q", got, "b")
	}
}

func TestGetAppName(t *testing.T) {
	t.Setenv("X_APP_NAME", "foo")

	r := decodeDump(t, serve(t, httptest.NewRequest(http.MethodGet, "/", nil)))

	if got := r.Headers["X-Application-Name"]; got != "foo" {
		t.Errorf("headers[X-Application-Name] = %q, want %q", got, "foo")
	}
}

func TestGetUncleanPath(t *testing.T) {
	r := decodeDump(t, serve(t, httptest.NewRequest(http.MethodGet, "/../x", nil)))

	if r.Path != "/../x" {
		t.Errorf("path = %q, want %q", r.Path, "/../x")
	}
}

func TestOptions(t *testing.T) {
	rec := serve(t, httptest.NewRequest(http.MethodOptions, "/a", nil))

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Allow"); got != allowedMethods {
		t.Errorf("Allow = %q, want %q", got, allowedMethods)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestHead(t *testing.T) {
	rec := serve(t, httptest.NewRequest(http.MethodHead, "/a", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("Allow"); got != allowedMethods {
		t.Errorf("Allow = %q, want %q", got, allowedMethods)
	}
	if got, ok := rec.Header()["Content-Type"]; ok {
		t.Errorf("Content-Type = %q, want none", got)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestMethodNotAllowed(t *testing.T) {
	methods := []string{
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodConnect,
	}
	for _, m := range methods {
		t.Run(m, func(t *testing.T) {
			rec := serve(t, httptest.NewRequest(m, "/a", nil))

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
			}
			if got := rec.Header().Get("Allow"); got != allowedMethods {
				t.Errorf("Allow = %q, want %q", got, allowedMethods)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want %q", got, "application/json")
			}
			want := "{\"message\":\"Method Not Allowed\"}\n"
			if got := rec.Body.String(); got != want {
				t.Errorf("body = %q, want %q", got, want)
			}
		})
	}
}
