package httpapi

import (
	"fmt"
	"math"
	"net/http"
	"net/netip"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/auth"
)

const (
	sessionCookie = "nvm_session"
	// Batas panjang mencegah bcrypt dipaksa memproses input raksasa; bcrypt sendiri hanya memakai 72 byte.
	maxUsernameLen = 50
	maxPasswordLen = 72
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type sessionUser struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Username           string `json:"username"`
	Role               string `json:"role"`
	DivisionID         int64  `json:"division_id"`
	DivisionName       string `json:"division_name,omitempty"`
	MustChangePassword bool   `json:"must_change_password"`
}

func (s *Server) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian login tidak terbaca.")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Username dan password wajib diisi.")
		return
	}
	if len(req.Username) > maxUsernameLen || len(req.Password) > maxPasswordLen {
		writeError(c, http.StatusUnauthorized, CodeInvalidCredentials, msgInvalidCredentials)
		return
	}

	now := s.now()
	res, err := s.auth.Login(c.Request.Context(), req.Username, req.Password, clientAddr(c), now)
	if err != nil {
		s.internalError(c, err)
		return
	}

	switch res.Outcome {
	case auth.LoginInvalid:
		writeError(c, http.StatusUnauthorized, CodeInvalidCredentials, msgInvalidCredentials)
		return
	case auth.LoginLocked:
		minutes := int(math.Ceil(float64(res.RetryAfterSeconds) / 60))
		c.Header("Retry-After", strconv.Itoa(res.RetryAfterSeconds))
		writeError(c, http.StatusLocked, CodeAccountLocked,
			fmt.Sprintf("Akun dikunci sementara karena terlalu banyak percobaan gagal. Coba lagi dalam %d menit.", minutes))
		return
	}

	u := res.User
	token, expires, err := s.tokens.Issue(u.ID, string(u.Role), u.TokenVersion, now)
	if err != nil {
		s.internalError(c, err)
		return
	}
	s.setSessionCookie(c, token, int(expires.Sub(now).Seconds()))

	c.JSON(http.StatusOK, gin.H{"user": sessionUser{
		ID:                 u.ID,
		Name:               u.Name,
		Username:           u.Username,
		Role:               string(u.Role),
		DivisionID:         u.DivisionID,
		MustChangePassword: u.MustChangePassword,
	}})
}

func (s *Server) setSessionCookie(c *gin.Context, token string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		// Lax cukup karena semua perubahan data memakai POST/PUT/DELETE; ditambah cek Origin di langkah 5.
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cookieSecure,
	})
}

func clientAddr(c *gin.Context) *netip.Addr {
	addr, err := netip.ParseAddr(c.ClientIP())
	if err != nil {
		return nil
	}
	addr = addr.Unmap()
	return &addr
}
