package twitch

import (
	"time"
)

type RateLimiter struct {
	joinLimit int
	throttle  chan time.Time
}

const Unlimited = -1

func CreateDefaultRateLimiter() *RateLimiter {
	return createRateLimiter(20)
}

func CreateVerifiedRateLimiter() *RateLimiter {
	return createRateLimiter(2000)
}

func CreateUnlimitedRateLimiter() *RateLimiter {
	return createRateLimiter(Unlimited)
}

func createRateLimiter(limit int) *RateLimiter {
	return &RateLimiter{
		joinLimit: limit,
		throttle:  make(chan time.Time, 10),
	}
}

func (r *RateLimiter) Throttle() {
	if r.joinLimit == Unlimited {
		return
	}

	<-r.throttle
}

func (r *RateLimiter) Start() {
	if r.joinLimit == Unlimited {
		return
	}

	r.fillThrottle()

	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		r.fillThrottle()
	}
}

func (r *RateLimiter) fillThrottle() {
	for i := 0; i < r.joinLimit; i++ {
		r.throttle <- time.Now()
	}
}
