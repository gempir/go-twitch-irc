package twitch

import (
	"sync"
	"time"
)

type RateLimiter interface {
	// This will impact how go-twitch-irc groups joins together per IRC message
	GetJoinLimit() int
	Throttle(count int)
	isUnlimited() bool
}

type WindowRateLimiter struct {
	joinLimit int
	window    []time.Time
	mutex     sync.Mutex
}

const Unlimited = -1
const TwitchRateLimitWindow = 10 * time.Second

func CreateDefaultRateLimiter() *WindowRateLimiter {
	return createRateLimiter(20)
}

func CreateVerifiedRateLimiter() *WindowRateLimiter {
	return createRateLimiter(2000)
}

func CreateUnlimitedRateLimiter() *WindowRateLimiter {
	return createRateLimiter(Unlimited)
}

func createRateLimiter(limit int) *WindowRateLimiter {
	var window []time.Time

	return &WindowRateLimiter{
		joinLimit: limit,
		window:    window,
	}
}

func (r *WindowRateLimiter) GetJoinLimit() int {
	return r.joinLimit
}

func (r *WindowRateLimiter) Throttle(count int) {
	if r.joinLimit == Unlimited {
		return
	}
	r.mutex.Lock()
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
		r.mutex.Unlock()
		return
	}

	time.Sleep(time.Until(r.window[0].Add(TwitchRateLimitWindow).Add(time.Millisecond * 100)))
	r.mutex.Unlock()
	r.Throttle(count)
}

func (r *WindowRateLimiter) isUnlimited() bool {
	return r.joinLimit == Unlimited
}
