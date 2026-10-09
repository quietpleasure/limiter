package limiter

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Config struct {
	Interval          time.Duration `yaml:"interval"`
	Burst             int           `yaml:"burst"`
	ExeptionAddresses []string      `yaml:"exeption_addresses"`
}

type client struct {
	lastSeen time.Time
	limiter  *rate.Limiter
}

// idleTimeout — сколько времени храним лимитер для IP без новых запросов,
// прежде чем удалить его из карты и освободить память. Не связано с
// интервалом восполнения токенов: если использовать здесь r.i, лимитер
// сбрасывался бы до полного burst каждые r.i секунд даже для IP,
// непрерывно шлющего запросы.
const idleTimeout = 10 * time.Minute

var DEFAULT_EXEPTION_ADDRESSES []string = []string{"127.0.0.1", "0.0.0.0", "localhost", "::1"}

type RateLimiter struct {
	mu *sync.RWMutex

	ips map[string]*client

	r rate.Limit
	b int
	i time.Duration
}

// var DEFAULT_RATE_LIMITER *RateLimiter = New(time.Second, 25)

func DefaultRateLimiter() *RateLimiter {
	return &RateLimiter{
		ips: make(map[string]*client),
		mu:  &sync.RWMutex{},
		r:   rate.Every(time.Second),
		b:   25,
		i:   time.Second,
	}
}

func New(interval time.Duration, burst int) *RateLimiter {
	if interval == 0 || burst == 0 {
		return DefaultRateLimiter()
	}
	i := &RateLimiter{
		ips: make(map[string]*client),
		mu:  &sync.RWMutex{},
		r:   rate.Every(interval),
		b:   burst,
		i:   interval,
	}

	return i
}

func (r *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	r.mu.Lock()
	client, exists := r.ips[ip]

	if !exists {
		r.mu.Unlock()
		return r.addIP(ip)
	}
	if time.Since(client.lastSeen) > idleTimeout {
		delete(r.ips, ip)
		r.mu.Unlock()
		return r.addIP(ip)
	}
	client.lastSeen = time.Now()
	limiter := client.limiter
	r.mu.Unlock()
	return limiter
}

func (r *RateLimiter) addIP(ip string) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()

	limiter := rate.NewLimiter(r.r, r.b)

	r.ips[ip] = &client{
		limiter:  limiter,
		lastSeen: time.Now(),
	}

	return limiter
}
