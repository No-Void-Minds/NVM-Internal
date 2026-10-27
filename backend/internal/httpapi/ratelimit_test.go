package httpapi

import (
	"testing"
	"time"
)

func TestLimiterAllowsBurstThenBlocks(t *testing.T) {
	l := newIPLimiter(30*time.Second, 10)
	now := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)

	for i := range 10 {
		if wait := l.allow("10.0.0.1", now); wait != 0 {
			t.Fatalf("percobaan ke-%d ditolak, padahal masih dalam batas", i+1)
		}
	}
	wait := l.allow("10.0.0.1", now)
	if wait != 30*time.Second {
		t.Fatalf("percobaan ke-11 harus menunggu 30 detik, dapat %v", wait)
	}
}

func TestLimiterRefillsOverTime(t *testing.T) {
	l := newIPLimiter(30*time.Second, 10)
	now := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	for range 10 {
		l.allow("10.0.0.1", now)
	}

	if wait := l.allow("10.0.0.1", now.Add(29*time.Second)); wait == 0 {
		t.Fatal("sebelum 30 detik belum boleh ada token baru")
	}
	if wait := l.allow("10.0.0.1", now.Add(30*time.Second)); wait != 0 {
		t.Fatalf("setelah 30 detik satu percobaan harus boleh, dapat tunggu %v", wait)
	}
}

func TestLimiterIsPerIP(t *testing.T) {
	l := newIPLimiter(30*time.Second, 10)
	now := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	for range 11 {
		l.allow("10.0.0.1", now)
	}
	if wait := l.allow("10.0.0.2", now); wait != 0 {
		t.Fatal("IP lain tidak boleh ikut terhambat")
	}
}

func TestLimiterSweepsIdleBuckets(t *testing.T) {
	l := newIPLimiter(30*time.Second, 10)
	now := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	l.allow("10.0.0.1", now)
	l.allow("10.0.0.2", now.Add(6*time.Minute))
	if _, ok := l.buckets["10.0.0.1"]; ok {
		t.Fatal("bucket yang sudah lama tidak dipakai harus dibuang")
	}
}
