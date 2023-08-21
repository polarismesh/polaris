package api

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestBaseCache_LastFetchTime(t *testing.T) {
	lt := time.Now()
	bc := &BaseCache{
		lastFetchTime: lt.Unix(),
	}
	assert.Equal(t, lt.Add(DefaultTimeDiff).Unix(), bc.LastFetchTime().Unix())
}
