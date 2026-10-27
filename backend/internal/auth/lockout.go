package auth

import (
	"math"
	"time"
)

// MaxFailedAttempts adalah jumlah gagal berturut-turut sebelum akun dikunci (FR-AUTH-09).
const MaxFailedAttempts = 5

// Durasi per tahap kuncian; indeks = Level - 1. Tahap setelah 30 menit tetap 30 menit.
var lockDurations = []time.Duration{5 * time.Minute, 15 * time.Minute, 30 * time.Minute}

// LockState mencerminkan kolom failed_login_count, lockout_level, dan locked_until di tabel users.
type LockState struct {
	FailedCount int
	Level       int
	LockedUntil time.Time // nilai nol berarti tidak ada kuncian
}

func (s LockState) IsLocked(now time.Time) bool {
	return !s.LockedUntil.IsZero() && now.Before(s.LockedUntil)
}

// RetryAfterSeconds dibulatkan ke atas supaya pengguna tidak dipersilakan mencoba sedetik terlalu cepat.
func (s LockState) RetryAfterSeconds(now time.Time) int {
	if !s.IsLocked(now) {
		return 0
	}
	return int(math.Ceil(s.LockedUntil.Sub(now).Seconds()))
}

// RegisterFailure mencatat satu login gagal. Nilai kedua true bila kegagalan ini memicu kuncian.
// Hitungan gagal dikosongkan saat dikunci, sedangkan Level dipertahankan sampai login berhasil,
// sehingga 5 gagal berikutnya setelah kunci berakhir naik ke tahap selanjutnya.
func RegisterFailure(s LockState, now time.Time) (LockState, bool) {
	s.FailedCount++
	if s.FailedCount < MaxFailedAttempts {
		return s, false
	}
	if s.Level < len(lockDurations) {
		s.Level++
	}
	s.FailedCount = 0
	s.LockedUntil = now.Add(lockDurations[s.Level-1])
	return s, true
}

// RegisterSuccess mengembalikan hitungan dan tahap kuncian ke awal (FR-AUTH-09).
func RegisterSuccess() LockState {
	return LockState{}
}