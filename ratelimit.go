package twitch

import (
	"fmt"
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

func (r *RateLimiter) isUnlimited() bool {
	return r.joinLimit == Unlimited
}

func (r *RateLimiter) Start() {
	if r.joinLimit == Unlimited {
		return
	}

	r.fillBucket()

	ticker := time.NewTicker(TwitchRateLimitWindow)
	for range ticker.C {
		r.fillBucket()
	}
}

func (r *RateLimiter) fillBucket() {
	fmt.Printf("%s Filling bucket %d\n", time.Now(), len(r.throttle))

	fillSize := r.joinLimit - len(r.throttle)
	for i := 0; i < fillSize; i++ {
		select {
		case r.throttle <- time.Now():
		default:
			fmt.Printf("%s Bucket is full\n", time.Now())
			// discard
		}
	}
}
