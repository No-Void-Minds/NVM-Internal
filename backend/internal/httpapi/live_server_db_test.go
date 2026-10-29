package httpapi

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/config"
	"nvm-internal/internal/testdb"
)

// liveServer menjalankan router sungguhan di atas database test, lalu dipanggil seperti browser:
// login lewat cookie, mengirim header Origin, dan membaca kode status. Dengan begitu pemasangan
// RBAC di router ikut teruji, bukan hanya logika service.
type liveServer struct {
	srv      *httptest.Server
	pool     *pgxpool.Pool
	division int64
}

func startServer(t *testing.T) liveServer {
	t.Helper()
	gin.SetMode(gin.TestMode)
	pool := testdb.Pool(t)
	division := testdb.Division(t, pool, "Marketing", true)
	cfg := config.Config{AppEnv: "test", JWTSecret: strings.Repeat("x", 32), StorageDir: t.TempDir()}
	r, err := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), pool, cfg)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return liveServer{srv: srv, pool: pool, division: division}
}

func (s liveServer) call(t *testing.T, c *http.Client, method, path, body string) int {
	t.Helper()
	return s.sendJSON(t, c, method, path, body, nil)
}

func (s liveServer) login(t *testing.T, username, password string) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	c := &http.Client{Jar: jar}
	body := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	if code := s.call(t, c, http.MethodPost, "/api/auth/login", body); code != http.StatusOK {
		t.Fatalf("login %s: status %d, mau 200", username, code)
	}
	return c
}
