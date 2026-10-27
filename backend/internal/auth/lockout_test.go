package auth

import (
	"testing"
	"time"
)

func failTimes(s LockState, now time.Time, n int) (LockState, bool) {
	var locked bool
	for range n {
		s, locked = RegisterFailure(s, now)
	}
	return s, locked
}

func TestFourFailuresDoNotLock(t *testing.T) {
	now := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	s, locked := failTimes(LockState{}, now, 4)
	if locked || s.IsLocked(now) {
		t.Fatal("4 kali gagal tidak boleh mengunci akun")
	}
	if s.FailedCount != 4 {
		t.Fatalf("FailedCount = %d, mau 4", s.FailedCount)
	}
}

func TestProgressiveLockDurations(t *testing.T) {
	now := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	want := []time.Duration{5 * time.Minute, 15 * time.Minute, 30 * time.Minute, 30 * time.Minute, 30 * time.Minute}

	s := LockState{}
	for i, d := range want {
		var locked bool
		s, locked = failTimes(s, now, MaxFailedAttempts)
		if !locked {
			t.Fatalf("tahap %d: gagal ke-5 harus mengunci akun", i+1)
		}
		if got := s.LockedUntil.Sub(now); got != d {
			t.Fatalf("tahap %d: durasi kunci %v, mau %v", i+1, got, d)
		}
		if !s.IsLocked(now.Add(d - time.Second)) {
			t.Fatalf("tahap %d: akun harus masih terkunci sedetik sebelum waktunya habis", i+1)
		}
		if s.IsLocked(now.Add(d)) {
			t.Fatalf("tahap %d: akun harus terbuka tepat saat waktunya habis", i+1)
		}
		now = now.Add(d)
	}
}

func TestSuccessResetsEverything(t *testing.T) {
	now := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	s, _ := failTimes(LockState{}, now, MaxFailedAttempts)
	s, _ = failTimes(s, now.Add(5*time.Minute), 3)

	s = RegisterSuccess()
	if s != (LockState{}) {
		t.Fatalf("login berhasil harus mengosongkan state, dapat %+v", s)
	}

	s, _ = failTimes(s, now, MaxFailedAttempts)
	if got := s.LockedUntil.Sub(now); got != 5*time.Minute {
		t.Fatalf("setelah reset, kuncian pertama harus 5 menit lagi, dapat %v", got)
	}
}

func TestRetryAfterRoundsUp(t *testing.T) {
	now := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	s := LockState{Level: 1, LockedUntil: now.Add(90*time.Second + 300*time.Millisecond)}
	if got := s.RetryAfterSeconds(now); got != 91 {
		t.Fatalf("RetryAfterSeconds = %d, mau 91", got)
	}
	if got := s.RetryAfterSeconds(now.Add(time.Hour)); got != 0 {
		t.Fatalf("setelah kunci habis RetryAfterSeconds = %d, mau 0", got)
	}
}