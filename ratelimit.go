package twitch

import (
	"fmt"
	"time"
)

type RateLimits struct {
	limit int32
	joins tAtomInt32
}

func CreateUnlimitedRateLimits() RateLimits {
	return RateLimits{
		limit: -1,
	}
}

func CreateDefaultRateLimits() RateLimits {
	return RateLimits{
		limit: 20,
	}
}

func CreateVerifiedRateLimits() RateLimits {
	return RateLimits{
		limit: 2000,
	}
}

func (r *RateLimits) Increment() {
	r.joins.increment()
	fmt.Println(r.joins.get())
}

func (r *RateLimits) Allowed() bool {
	if r.limit == -1 {
		return true
	}

	if r.joins.get() <= r.limit {
		return true
	}

	return false
}

func (r *RateLimits) StartFixedWindowLimiter() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		fmt.Println("resetting joins")
		r.joins.set(0)
	}
}
