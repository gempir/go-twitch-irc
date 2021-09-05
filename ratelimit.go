package twitch

import (
	"time"

	"golang.org/x/time/rate"
)

type RateLimits struct {
	joinLimiter *rate.Limiter
}

func CreateUnlimitedRateLimits() RateLimits {
	return RateLimits{
		joinLimiter: rate.NewLimiter(rate.Inf, 0),
	}
}

func CreateDefaultRateLimits() RateLimits {
	return RateLimits{
		joinLimiter: rate.NewLimiter(rate.Every(time.Second*10/20), 1),
	}
}

func CreateVerifiedRateLimits() RateLimits {
	return RateLimits{
		joinLimiter: rate.NewLimiter(rate.Every(time.Second*10/2000), 1),
	}
}
