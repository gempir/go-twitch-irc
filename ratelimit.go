package twitch

import (
	"time"
)

type RateLimiter struct {
	joinLimit int
	throttle  chan time.Time
}

const Unlimited = -1
const TwitchRateLimitWindow = 10 * time.Second

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

	tickerTime := TwitchRateLimitWindow / time.Duration(r.joinLimit)

	ticker := time.NewTicker(tickerTime)
	for range ticker.C {
		select {
		case r.throttle <- time.Now():
		default:
			// discard
		}
	}
}
