package strategy

type RateLimiterStrategy interface {
	// Check reports an infrastructure error separately from quota exhaustion.
	Check(ip string, token string) (blocked bool, err error)
	CheckIsBlocked(ip string, token string) bool
}
