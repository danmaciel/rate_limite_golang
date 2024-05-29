package strategy

type RateLimiterStrategy interface {
	CheckIsBlocked(ip string, token string) bool
}
