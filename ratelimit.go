package twitch

type RateLimits struct {
	JoinsAndPartsPerTenSeconds int
}

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
