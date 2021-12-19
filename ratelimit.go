package twitch

import (
	"time"
)

type RateLimiter struct {
	joinLimit int
	window    []time.Time
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
	var window []time.Time

	return &RateLimiter{
		joinLimit: limit,
		window:    window,
	}
}

func (r *RateLimiter) Throttle(count int) {
	if r.joinLimit == Unlimited {
		return
	}

	newWindow := []time.Time{}

	for i := 0; i < len(r.window); i++ {
		if r.window[i].Add(TwitchRateLimitWindow).After(time.Now()) {
			newWindow = append(newWindow, r.window[i])
		}
	}

	if r.joinLimit-len(newWindow) > count || len(newWindow) == 0 {
		for i := 0; i < count; i++ {
			newWindow = append(newWindow, time.Now())
		}
		r.window = newWindow
		return
	}

	time.Sleep(time.Until(r.window[0].Add(TwitchRateLimitWindow).Add(time.Millisecond * 100)))
	r.Throttle(count)
}

func (r *RateLimiter) isUnlimited() bool {
	return r.joinLimit == Unlimited
}
