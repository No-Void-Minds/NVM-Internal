package httpapi

import (
	"sync"
	"time"
)

// ipLimiter adalah token bucket per alamat IP. Disimpan di memori karena hanya ada satu
// instance backend; hitungan kembali kosong saat server restart, dan itu dapat diterima.
type ipLimiter struct {
	mu        sync.Mutex
	every     time.Duration
	burst     float64
	buckets   map[string]*bucket
	lastSweep time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

func newIPLimiter(every time.Duration, burst int) *ipLimiter {
	return &ipLimiter{every: every, burst: float64(burst), buckets: map[string]*bucket{}}
}

// allow mengembalikan 0 bila request boleh lewat, atau lama tunggu sampai satu token tersedia.
func (l *ipLimiter) allow(ip string, now time.Time) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.sweep(now)

	b, ok := l.buckets[ip]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[ip] = b
	}

	refill := float64(now.Sub(b.last)) / float64(l.every)
	b.tokens = min(l.burst, b.tokens+refill)
	b.last = now

	if b.tokens >= 1 {
		b.tokens--
		return 0
	}
	return time.Duration((1 - b.tokens) * float64(l.every))
}

// sweep membuang bucket yang sudah penuh kembali, supaya map tidak tumbuh terus.
func (l *ipLimiter) sweep(now time.Time) {
	full := time.Duration(l.burst) * l.every
	if now.Sub(l.lastSweep) < full {
		return
	}
	for ip, b := range l.buckets {
		if now.Sub(b.last) >= full {
			delete(l.buckets, ip)
		}
	}
	l.lastSweep = now
}
