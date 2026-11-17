package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SessionMaxAge adalah umur mutlak token, cookie, dan baris sesi sejak login. Sejak revisi 8 ini satu-satunya
// batas waktu sesi: idle timeout 30 menit dihapus atas permintaan Manajer, jadi sesi yang ditinggalkan tetap
// hidup sampai 9 jam. last_seen_at masih dicatat, tetapi hanya sebagai keterangan, bukan penentu.
const SessionMaxAge = 9 * time.Hour

var ErrInvalidToken = errors.New("token tidak valid")

// Claims sengaja ringkas. Role ikut disimpan sebagai informasi, tetapi hak akses selalu
// dibaca ulang dari database supaya perubahan peran langsung berlaku.
type Claims struct {
	Role         string `json:"role"`
	TokenVersion int32  `json:"tv"`
	// SessionID menunjuk baris user_sessions; token tanpa sid (sebelum migrasi 000006) ditolak.
	SessionID string `json:"sid"`
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

func (t TokenIssuer) Issue(userID int64, role string, tokenVersion int32, sessionID string, now time.Time) (string, time.Time, error) {
	expires := now.Add(SessionMaxAge)
	claims := Claims{
		Role:         role,
		TokenVersion: tokenVersion,
		SessionID:    sessionID,
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
