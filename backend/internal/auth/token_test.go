package auth

import (
	"strings"
	"testing"
	"time"
)

const testSecret = "rahasia-uji-yang-panjangnya-lebih-dari-32"

func TestTokenRoundTrip(t *testing.T) {
	issuer, err := NewTokenIssuer(testSecret)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)

	tok, exp, err := issuer.Issue(7, "staff", 3, "sid-uji", now)
	if err != nil {
		t.Fatal(err)
	}
	if exp.Sub(now) != SessionMaxAge {
		t.Fatalf("umur token %v, mau %v", exp.Sub(now), SessionMaxAge)
	}

	claims, userID, err := issuer.Parse(tok, now.Add(29*time.Minute))
	if err != nil {
		t.Fatalf("token yang masih berlaku ditolak: %v", err)
	}
	if userID != 7 || claims.TokenVersion != 3 || claims.Role != "staff" || claims.SessionID != "sid-uji" {
		t.Fatalf("isi token salah: userID=%d claims=%+v", userID, claims)
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	issuer, _ := NewTokenIssuer(testSecret)
	now := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	tok, _, _ := issuer.Issue(7, "staff", 0, "sid-uji", now)

	if _, _, err := issuer.Parse(tok, now.Add(SessionMaxAge+time.Second)); err == nil {
		t.Fatal("token yang sudah lewat umur mutlaknya harus ditolak")
	}
}

func TestTokenFromOtherSecretRejected(t *testing.T) {
	now := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	other, _ := NewTokenIssuer(strings.Repeat("z", 40))
	tok, _, _ := other.Issue(7, "admin", 0, "sid-uji", now)

	issuer, _ := NewTokenIssuer(testSecret)
	if _, _, err := issuer.Parse(tok, now); err == nil {
		t.Fatal("token yang ditandatangani kunci lain harus ditolak")
	}
}

func TestUnsignedTokenRejected(t *testing.T) {
	// Header {"alg":"none","typ":"JWT"} dan payload {"sub":"1","tv":0,"exp":9999999999}, tanpa tanda tangan.
	tok := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiIxIiwidHYiOjAsImV4cCI6OTk5OTk5OTk5OX0."
	issuer, _ := NewTokenIssuer(testSecret)
	if _, _, err := issuer.Parse(tok, time.Now()); err == nil {
		t.Fatal("token dengan alg none harus ditolak")
	}
}

func TestShortSecretRejected(t *testing.T) {
	if _, err := NewTokenIssuer("pendek"); err == nil {
		t.Fatal("JWT_SECRET pendek harus ditolak saat server mulai")
	}
}
