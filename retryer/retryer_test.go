package retryer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetWaitTime(t *testing.T) {

	testCases := []struct {
		retryCount int
		waitTime   time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, 1600 * time.Millisecond},
	}

	for _, testCase := range testCases {
		waitTime := getWaitTime(testCase.retryCount)
		assert.Equal(t, testCase.waitTime, waitTime)
	}
}
