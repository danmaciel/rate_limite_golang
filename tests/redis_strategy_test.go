package tests

import (
	"testing"
	"time"

	"github.com/danmaciel/rate_limite_golang/internal/entity"
	"github.com/danmaciel/rate_limite_golang/internal/infra/persistence"
	"github.com/stretchr/testify/assert"
)

func TestRedisStrategy(t *testing.T) {

	p := persistence.NewRedisStrategy(":6379", "", 2)

	errDelete := p.Delete("aaa")

	assert.Nil(t, errDelete)

	dateNow := time.Now().Format(time.RFC3339)
	eInsert := p.Set("aaa", entity.DataRateLimiter{Count: 10, TimeExec: dateNow})
	assert.Nil(t, eInsert)

	var data2 entity.DataRateLimiter
	e := p.Get("aaa", &data2)
	assert.Nil(t, e)
	assert.NotNil(t, data2)
	assert.Equal(t, data2.Count, 10)
	assert.Equal(t, data2.TimeExec, dateNow)

}
