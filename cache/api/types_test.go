package api

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestBaseCache_LastFetchTime(t *testing.T) {
	bc := &BaseCache{}
	assert.EqualValues(t, 0, bc.LastFetchTime().Unix())

	lt := time.Now()
	bc.lastFetchTime = lt.Unix()
	assert.Equal(t, lt.Add(DefaultTimeDiff).Unix(), bc.LastFetchTime().Unix())
}
