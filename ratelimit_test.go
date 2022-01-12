package twitch

import (
	"sync"
	"testing"
	"time"
)

func TestUnlimitedRateLimiter(t *testing.T) {
	limiter := CreateUnlimitedRateLimiter()

	if !limiter.IsUnlimited() {
		t.Fatal("Limited must be unlimited")
	}

	for i := 0; i < 5000; i++ {
		before := time.Now()
		limiter.Throttle(1)
		after := time.Now()

		diff := after.Sub(before)

		if diff >= windowRateLimiterSleepDuration {
			t.Fatal("No sleeping allowed in unlimited rate limiter")
		}
	}
}

func TestDefaultRateLimiter(t *testing.T) {
	limiter := CreateDefaultRateLimiter()

	if limiter.IsUnlimited() {
		t.Fatal("Limited must not be unlimited")
	}

	for i := 0; i < 20; i++ {
		before := time.Now()
		limiter.Throttle(1)
		after := time.Now()

		diff := after.Sub(before)

		if diff >= windowRateLimiterSleepDuration {
			t.Fatal("No sleeping should take place while we're within the rate limit")
		}
	}
}

func TestDefaultRateLimiterBigChunks(t *testing.T) {
	limiter := CreateDefaultRateLimiter()

	if limiter.IsUnlimited() {
		t.Fatal("Limited must not be unlimited")
	}

	for i := 0; i < 2; i++ {
		before := time.Now()
		limiter.Throttle(10)
		after := time.Now()

		diff := after.Sub(before)

		if diff >= windowRateLimiterSleepDuration {
			t.Fatal("No sleeping should take place while we're within the rate limit")
		}
	}
}

func TestDefaultRateLimiterMultiThread(t *testing.T) {
	limiter := CreateDefaultRateLimiter()

	wg := sync.WaitGroup{}

	sender := func(n int) {
		for i := 0; i < n; i++ {
			limiter.Throttle(1)
		}
	}

	before := time.Now()
	numWorkers := 2
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			sender(10)
			wg.Done()
		}()
	}
	wg.Wait()
	after := time.Now()
	diff := after.Sub(before)

	if diff >= windowRateLimiterSleepDuration {
		t.Fatal("No sleeping allowed in unlimited rate limiter")
	}
}
