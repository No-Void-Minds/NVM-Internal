package httpapi

import (
	"github.com/gin-gonic/gin"
)

// Frontend memilih tampilan berdasarkan kode; teks pesan boleh berubah, kode jangan.
const (
	CodeBadRequest         = "bad_request"
	CodeUnauthorized       = "unauthorized"
	CodeForbidden          = "forbidden"
	CodeNotFound           = "not_found"
	CodeConflict           = "conflict"
	CodeInternal           = "internal"
	CodeUnavailable        = "service_unavailable"
	CodeInvalidCredentials = "invalid_credentials"
	CodeAccountLocked      = "account_locked"
)

// Satu pesan untuk username tidak ada, password salah, dan akun nonaktif (AC-01).
const msgInvalidCredentials = "Username atau password salah."

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorBody struct {
	Error apiError `json:"error"`
}

func writeError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, errorBody{Error: apiError{Code: code, Message: message}})
}

// internalError mencatat detail teknis ke log dan hanya mengirim pesan umum ke pengguna (NFR-REL-06).
func (s *Server) internalError(c *gin.Context, err error) {
	s.log.Error("galat internal",
		"err", err,
		"method", c.Request.Method,
		"route", c.FullPath(),
		"request_id", requestID(c),
	)
	writeError(c, 500, CodeInternal, "Terjadi kesalahan di server. Coba lagi beberapa saat.")
}
