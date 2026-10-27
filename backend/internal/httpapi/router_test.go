package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/config"
)

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.Config{AppEnv: "test", JWTSecret: strings.Repeat("x", 32)}
	r, err := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestUnknownRouteReturnsJSONError(t *testing.T) {
	r := newTestRouter(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/tidak-ada", nil))

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, mau 404", w.Code)
	}
	var body errorBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body bukan JSON: %v", err)
	}
	if body.Error.Code != CodeNotFound || body.Error.Message == "" {
		t.Fatalf("body galat tidak sesuai format: %+v", body)
	}
	if w.Header().Get("X-Request-ID") == "" {
		t.Fatal("header X-Request-ID kosong")
	}
}
