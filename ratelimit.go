package twitch

import (
	"time"
)

type RateLimits struct {
	limit    int
	throttle chan time.Time
}

func CreateUnlimitedRateLimits() *RateLimits {
	return CreateRateLimits(-1)
}

func CreateDefaultRateLimits() *RateLimits {
	return CreateRateLimits(20)
}

func CreateVerifiedRateLimits() *RateLimits {
	return CreateRateLimits(2000)
}

func CreateRateLimits(limit int) *RateLimits {
	return &RateLimits{
		limit:    limit,
		throttle: make(chan time.Time, 10),
	}
}

func (r *RateLimits) StartRateLimiter() {
	r.fillThrottle()

	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		r.fillThrottle()
	}
}

func (r *RateLimits) fillThrottle() {
	for i := 0; i < r.limit; i++ {
		r.throttle <- time.Now()
	}
}
