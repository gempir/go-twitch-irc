package twitch

import (
	"time"

	"golang.org/x/time/rate"
)

type RateLimits struct {
	limiter *rate.Limiter
}

// move these later? not sure where to put them
const (
	MessagePrefixJoin = "JOIN"
)

func CreateDefaultRateLimits() RateLimits {
	return RateLimits{
		limiter: rate.NewLimiter(rate.Every(time.Second*10/20), 1),
	}
}

func CreateVerifiedRateLimits() RateLimits {
	return RateLimits{
		limiter: rate.NewLimiter(rate.Every(time.Second*10/100), 1), // verify limits later
	}
}
