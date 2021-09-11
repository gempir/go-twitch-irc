package twitch

import (
	"time"
)

type RateLimits struct {
	limit    int32
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

func CreateRateLimits(limit int32) *RateLimits {
	return &RateLimits{
		limit:    limit,
		throttle: make(chan time.Time, 10),
	}
}

func (r *RateLimits) StartRateLimiter() {
	ticker := time.NewTicker((10 * time.Second) / time.Duration(r.limit))
	for t := range ticker.C {
		r.throttle <- t
	}
}
