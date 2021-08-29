package twitch

import (
	"fmt"
	"time"
)

type RateLimits struct {
	JoinsAndPartsPerTenSeconds int32
	joins                      tAtomInt32
}

// move these later? not sure where to put them
const (
	MessagePrefixJoin = "JOIN"
)

func CreateDefaultRateLimits() RateLimits {
	return RateLimits{
		JoinsAndPartsPerTenSeconds: 20,
	}
}

func CreateVerifiedRateLimits() RateLimits {
	return RateLimits{
		JoinsAndPartsPerTenSeconds: 20,
	}
}

func (r *RateLimits) StartRateLimitTicker() {
	go func() {
		for {
			// i don't know why this results in 20s and that's why I have to divide it, figure it out later
			// fmt.Println(time.Duration(r.JoinsAndPartsPerTenSeconds) * time.Second)
			time.Sleep((time.Second * time.Duration(r.JoinsAndPartsPerTenSeconds)) / 2)
			fmt.Println("reset ratelimit")
			r.joins.set(0)
		}
	}()
}
