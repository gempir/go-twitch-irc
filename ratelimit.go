package twitch

import (
	"time"

	"golang.org/x/time/rate"
)

type RateLimits struct {
	joinLimiter *rate.Limiter
}

func CreateDefaultRateLimits() RateLimits {
	return RateLimits{
		joinLimiter: rate.NewLimiter(rate.Every(time.Second*10/20), 1),
	}
}

func CreateVerifiedRateLimits() RateLimits {
	return RateLimits{
		joinLimiter: rate.NewLimiter(rate.Every(time.Second*10/100), 1), // verify limits later
	}
}
