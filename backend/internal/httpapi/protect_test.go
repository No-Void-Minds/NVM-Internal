package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func postLogin(t *testing.T, origin string) *httptest.ResponseRecorder {
	t.Helper()
	r := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/api/auth/login", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestPostWithoutOriginRejected(t *testing.T) {
	if w := postLogin(t, ""); w.Code != http.StatusForbidden {
		t.Fatalf("POST tanpa Origin: status %d, mau 403", w.Code)
	}
}

func TestPostFromOtherSiteRejected(t *testing.T) {
	if w := postLogin(t, "http://situs-lain.example"); w.Code != http.StatusForbidden {
		t.Fatalf("POST dari situs lain: status %d, mau 403", w.Code)
	}
}

func TestPostFromSameOriginPassesThrough(t *testing.T) {
	// Body kosong sengaja dipakai supaya handler berhenti di validasi (400) tanpa menyentuh database.
	if w := postLogin(t, "http://127.0.0.1:8080"); w.Code != http.StatusBadRequest {
		t.Fatalf("POST dari origin yang sama: status %d, mau 400 dari validasi isian", w.Code)
	}
}

func TestLoginRateLimitedPerIP(t *testing.T) {
	r := newTestRouter(t)
	var last int
	for range 11 {
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/api/auth/login", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://127.0.0.1:8080")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		last = w.Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("percobaan ke-11 dari IP yang sama: status %d, mau 429", last)
	}
}
