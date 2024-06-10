package entity

type DataRateLimiter struct {
	Count       int
	TimeExec    string
	InBlackList bool
}
