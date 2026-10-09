package limiter

import (
	"testing"
	"time"
)

// TestRateLimiter_SustainedRate проверяет, что после начального burst
// лимитер не выдаёт одному и тому же IP больше burst разрешений в секунду
// при непрерывном потоке запросов.
//
// Регрессия на баг: client.lastSeen обновлялся только при создании записи,
// поэтому ровно через interval после первого запроса запись удалялась из
// карты и на следующий вызов пересоздавалась с чистым burst — непрерывно
// долбящий IP получал полный burst заново каждую секунду вместо ожидаемого
// "burst один раз, потом не больше 1 запроса/сек".
func TestRateLimiter_SustainedRate(t *testing.T) {
	const (
		interval = time.Second
		burst    = 5
		duration = 4 * time.Second
		ip       = "203.0.113.42"
	)

	rl := New(interval, burst)

	start := time.Now()
	allowedPerSecond := make([]int, int(duration/time.Second)+1)

	for time.Since(start) < duration {
		sec := int(time.Since(start) / time.Second)
		if rl.GetLimiter(ip).Allow() {
			allowedPerSecond[sec]++
		}
	}

	t.Logf("allowed per second: %v", allowedPerSecond)

	if allowedPerSecond[0] < burst {
		t.Errorf("ожидался начальный burst >= %d на 0-й секунде, получено %d", burst, allowedPerSecond[0])
	}

	for sec := 1; sec < len(allowedPerSecond); sec++ {
		if allowedPerSecond[sec] > burst {
			t.Errorf("секунда %d: разрешено %d запросов (> burst=%d) — лимитер сброшен до полного burst повторно, баг не исправлен", sec, allowedPerSecond[sec], burst)
		}
	}
}
