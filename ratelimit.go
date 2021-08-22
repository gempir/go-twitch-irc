package twitch

import (
	"container/list"
	"context"
	"time"
)

//startMiddlewares initializes the tracker tickers/contexts and middlewares.
func (c *Client) startMiddlewares() {

	//guard agains multiple middlewares.
	c.stopMiddlewares()

	c.whisperLimiter.startTracker(c.whisperRatelimit)
	go c.whisperMiddleware()
	c.authLimiter.startTracker(c.authRateLimit)
	go c.authMiddleware()
	c.joinLimiter.startTracker(c.joinRateLimit)
	go c.joinMiddleware()

	for channel, tracker := range c.sayLimiters {
		if tracker.isMod {
			tracker.startTracker(c.sayModRateLimit)
		} else {
			tracker.startTracker(c.sayRateLimit)
		}
		go c.sayMiddleware(channel)
	}
}

//SetWhisperRateLimit sets the ratelimit of whispers.
func (c *Client) SetWhisperRateLimit(rateTime time.Duration) {
	c.whisperRatelimit = rateTime
	c.whisperLimiter.ticker.Reset(rateTime)
}

//SetJoinRateLimit updates the ratelimit of joins.
func (c *Client) SetJoinRateLimit(rateTime time.Duration) {
	c.joinRateLimit = rateTime
	c.joinLimiter.ticker.Reset(rateTime)
}

//SetAuthRateLimit updates the rate limit of Authentication Requests.
func (c *Client) SetAuthRateLimit(rateTime time.Duration) {
	c.authRateLimit = rateTime
	c.authLimiter.ticker.Reset(rateTime)
}

//SetSayRateLimit updates the moderator/operator and unknown rate limits for Say requests.
func (c *Client) SetSayRateLimit(modRateTime, unknownRateTime time.Duration) {
	c.sayModRateLimit = modRateTime
	c.sayRateLimit = unknownRateTime
	for _, tracker := range c.sayLimiters {
		if tracker.isMod {
			tracker.ticker.Reset(modRateTime)
		} else {
			tracker.ticker.Reset(unknownRateTime)
		}
	}
}

//SetIgnoreRateLimitsBot sets the bot to ignore all ratelimits. This includes SAY rate limits.
func (c *Client) SetIgnoreRateLimitsBot() {
	c.SetWhisperRateLimit(IgnoreRatelimit)
	c.SetAuthRateLimit(IgnoreRatelimit)
	c.SetJoinRateLimit(IgnoreRatelimit)
	c.SetSayRateLimit(IgnoreRatelimit, IgnoreRatelimit)
}

//SetVerifiedBot updates all ratelimits to use those of a verified bot. This includes SAY rate limits.
func (c *Client) SetVerifiedBot() {
	c.SetWhisperRateLimit(VerifiedBotWhisper)
	c.SetAuthRateLimit(VerifiedBotAuth)
	c.SetJoinRateLimit(VerifiedBotJoin)
	c.SetSayRateLimit(ModeratorBotSay, UnknownBotSay)
}

//SetKnownBot updates all rate limits to use those of a known bot. This includes SAY rate limits.
func (c *Client) SetKnownBot() {
	c.SetWhisperRateLimit(KnownBotWhisper)
	c.SetAuthRateLimit(KnownBotAuth)
	c.SetJoinRateLimit(KnownBotJoin)
	c.SetSayRateLimit(ModeratorBotSay, UnknownBotSay)
}

//SetUnknownBot updates all rate limits to use those of an unknown bot. This includes SAY rate limits.
func (c *Client) SetUnknownBot() {
	c.SetWhisperRateLimit(UnknownBotWhisper)
	c.SetAuthRateLimit(UnknownBotAuth)
	c.SetJoinRateLimit(UnknownBotJoin)
	c.SetSayRateLimit(ModeratorBotSay, UnknownBotSay)
}

//BecomeModded Updates the input channel to have the correct mod status and updates the ticker respectively.
func (c *Client) BecomeModded(channel string) { //TODO use somewhere.
	c.sayLimiters[channel].isMod = true
	c.sayLimiters[channel].ticker.Reset(c.sayModRateLimit)
}

//BecomeUnModded Updates the input channel to have the correct mod status and updates the ticker respectively.
func (c *Client) BecomeUnModded(channel string) { //TODO use somewhere.
	c.sayLimiters[channel].isMod = false
	c.sayLimiters[channel].ticker.Reset(c.sayRateLimit)
}

//stopMiddlewares stops the middlewares for all of the middlewares. Use when disconnected to stop tickers.
func (c *Client) stopMiddlewares() {

	c.whisperLimiter.stopTracker()
	c.authLimiter.stopTracker()
	c.joinLimiter.stopTracker()

	for _, tracker := range c.sayLimiters {

		tracker.stopTracker()
	}
}

//sayMiddleware tracks the state of the middleware on a channel.
//when a say is sent to a channel, it is intercepted by the first case. if the rate limit is enabled,
//the message is sent to the queue. if it is not enabled, it is pushed directly to client.send()
//a new message on pushing to the queue checks if the queue is disabled, and if it is, reinables and restarts it.
func (c *Client) sayMiddleware(channel string) {
	r, ok := c.sayLimiters[channel]

	if !ok {
		// there was an error here and it must have departed before joining.
		return
	}
	messageChan := r.messageChannel
	for {
		select {
		case message := <-messageChan:
			if r.isMod {
				if c.sayModRateLimit != IgnoreRatelimit {
					if !r.tickerRunning {
						r.ticker.Reset(c.sayModRateLimit)
						r.tickerRunning = true
						//TODO add mutex to prevent perpetual lockout if keep resetting the timer?
						//is this safe because only one middleware is active at once?
						//may need to add lock to ratetimeLimiter?
					}
					r.messages.PushBack(message)
				} else {
					c.send(message)
				}
			} else {
				if c.sayRateLimit != IgnoreRatelimit {
					if !r.tickerRunning {
						r.ticker.Reset(c.sayRateLimit)
						r.tickerRunning = true
					}
					r.messages.PushBack(message)
				} else {
					c.send(message)
				}
			}
		case <-r.messageContext.Done():
			//empty queue
			r.messages.Init()
			//turn off ticker,
			r.ticker.Stop()
			r.tickerRunning = false
			//no longer listening for user.
			return
		case <-r.ticker.C:

			potentialMessage := r.messages.Front()
			if potentialMessage == nil {
				r.ticker.Stop()
				r.tickerRunning = false
				continue
			}
			r.messages.Remove(potentialMessage)
			message, ok := potentialMessage.Value.(string)
			if !ok {
				continue
			}
			c.send(message)

		}
	}

}

func (c *Client) whisperMiddleware() {
	r := c.whisperLimiter
	for {
		select {
		case message := <-r.messageChannel:
			if c.whisperRatelimit != IgnoreRatelimit {
				if !r.tickerRunning {
					r.ticker.Reset(c.whisperRatelimit)
					r.tickerRunning = true
				}
				r.messages.PushBack(message)
			} else {
				c.send(message)
			}
		case <-r.messageContext.Done():
			//empty queue
			r.messages.Init()
			//turn off ticker,
			r.ticker.Stop()
			r.tickerRunning = false
			//no longer listening for user.
			return
		case <-r.ticker.C:
			potentialMessage := r.messages.Front()
			if potentialMessage == nil {
				r.ticker.Stop()
				r.tickerRunning = false
				continue
			}
			r.messages.Remove(potentialMessage)
			message, ok := potentialMessage.Value.(string)
			if !ok {
				continue
			}
			c.send(message)
		}
	}
}

func (c *Client) joinMiddleware() {
	r := c.joinLimiter
	for {
		select {
		case message := <-r.messageChannel:
			if c.joinRateLimit != IgnoreRatelimit {
				if !r.tickerRunning {
					r.ticker.Reset(c.joinRateLimit)
					r.tickerRunning = true
				}
				r.messages.PushBack(message)
			} else {
				c.send(message)
			}
		case <-r.messageContext.Done():
			//empty queue
			r.messages.Init()
			//turn off ticker,
			r.ticker.Stop()
			r.tickerRunning = false
			//no longer listening for user.
			return
		case <-r.ticker.C:
			potentialMessage := r.messages.Front()
			if potentialMessage == nil {
				r.ticker.Stop()
				r.tickerRunning = false
				continue
			}
			r.messages.Remove(potentialMessage)
			message, ok := potentialMessage.Value.(string)
			if !ok {
				continue
			}
			c.send(message)
		}
	}
}

func (c *Client) authMiddleware() {
	r := c.authLimiter
	for {
		select {
		case message := <-r.messageChannel:
			if c.authRateLimit != IgnoreRatelimit {
				if !r.tickerRunning {
					r.ticker.Reset(c.authRateLimit)
					r.tickerRunning = true
				}
				r.messages.PushBack(message)
			} else {
				c.send(message)
			}
		case <-r.messageContext.Done():
			//empty queue
			r.messages.Init()
			//turn off ticker,
			r.ticker.Stop()
			r.tickerRunning = false
			//no longer listening for user.
			return
		case <-r.ticker.C:
			potentialMessage := r.messages.Front()
			if potentialMessage == nil {
				r.ticker.Stop()
				r.tickerRunning = false
				continue
			}
			r.messages.Remove(potentialMessage)
			message, ok := potentialMessage.Value.(string)
			if !ok {
				continue
			}
			c.send(message)
		}
	}
}

//RateLimiter acts as a go-between for the messages sent to the channel, and the output.
//Each message is sent after each tick of a ticker.
//If no messages are sent after a time, the ticker is stopped.
type RateLimiter struct {
	messages             *list.List
	isMod                bool
	messageChannel       chan string
	messageContext       context.Context
	messageContextCancel context.CancelFunc
	ticker               *time.Ticker
	//todo add update time chan
	tickerRunning bool
}

func (t *RateLimiter) startTracker(rate time.Duration) {

	ctx, cancel := context.WithCancel(context.Background())
	t.messageContext = ctx
	t.messageContextCancel = cancel
	t.ticker = time.NewTicker(rate)
}

func (t *RateLimiter) stopTracker() {
	if t.messageContextCancel != nil {
		t.messageContextCancel()
	}
}
