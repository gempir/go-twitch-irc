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
	var throttle chan time.Time
	if limit == Unlimited {
		throttle = make(chan time.Time)
	} else {
		throttle = make(chan time.Time, limit)
	}

	return &RateLimiter{
		joinLimit: limit,
		throttle:  throttle,
	}
}

func (r *RateLimiter) Throttle(count int) {
	if r.joinLimit == Unlimited {
		return
	}

	for i := 0; i < count; i++ {
		<-r.throttle
	}
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
		select {
		case r.throttle <- time.Now():
		default:
			// discarding rest, throttle already filled
		}
	}
}
