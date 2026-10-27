package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SessionTTL adalah umur token sekaligus batas idle (FR-AUTH-05, NFR-SEC-02).
const SessionTTL = 30 * time.Minute

var ErrInvalidToken = errors.New("token tidak valid")

// Claims sengaja ringkas. Role ikut disimpan sebagai informasi, tetapi hak akses selalu
// dibaca ulang dari database supaya perubahan peran langsung berlaku.
type Claims struct {
	Role         string `json:"role"`
	TokenVersion int32  `json:"tv"`
	jwt.RegisteredClaims
}

type TokenIssuer struct {
	secret []byte
}

func NewTokenIssuer(secret string) (TokenIssuer, error) {
	if len(secret) < 32 {
		return TokenIssuer{}, errors.New("JWT_SECRET minimal 32 karakter")
	}
	return TokenIssuer{secret: []byte(secret)}, nil
}

func (t TokenIssuer) Issue(userID int64, role string, tokenVersion int32, now time.Time) (string, time.Time, error) {
	expires := now.Add(SessionTTL)
	claims := Claims{
		Role:         role,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	return signed, expires, err
}

// Parse hanya menerima HS256; tanpa batasan ini token dengan alg "none" atau algoritme lain bisa lolos.
func (t TokenIssuer) Parse(token string, now time.Time) (Claims, int64, error) {
	var c Claims
	_, err := jwt.ParseWithClaims(token, &c,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	if err != nil {
		return Claims{}, 0, ErrInvalidToken
	}
	userID, err := strconv.ParseInt(c.Subject, 10, 64)
	if err != nil {
		return Claims{}, 0, ErrInvalidToken
	}
	return c, userID, nil
}
