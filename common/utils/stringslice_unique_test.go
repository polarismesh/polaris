package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringSliceDeDuplication(t *testing.T) {
	s := []string{"1", "2", "", "_invalid", "1", "23"}
	s2 := StringSliceDeDuplication(s)
	assert.Equal(t, s2, []string{"1", "2", "", "_invalid", "23"})
}
