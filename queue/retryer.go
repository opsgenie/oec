package queue

import (
	"time"
	"net/http"
	"math"
	"github.com/pkg/errors"
	"net"
)

const timeout = 1 * time.Second
var tokenClient = &http.Client{Timeout: timeout}	// todo timeout decision
var httpGetMethod = tokenClient.Get

const maxWaitInterval = 5
const maxRetryCount = 30

var retryStatusCodes = map[int]struct{}{
	429: {},
	//408: {},
}

type getMethod func(url string) (*http.Response, error)

type Retryer struct {
	getMethod getMethod
}

func NewRetryer() *Retryer {
	retryer := &Retryer{
		getMethod: getWithExponentialBackoff,
	}
	return retryer
}

func shouldRetry(statusCode int) bool{
	_, shouldRetry := retryStatusCodes[statusCode]

	if (statusCode >= 500 && statusCode <= 599) || shouldRetry {
		return true
	}
	return false
}

func getWaitTime(retryCount int) time.Duration{
	waitTime := math.Pow(2, float64(retryCount)) * 100
	//waitTime = math.Min(waitTime, float64(maxWaitInterval)) // todo min value
	return time.Duration(waitTime) * time.Millisecond
}

func getWithExponentialBackoff(url string) (*http.Response, error) {

	for retryCount := 0 ; retryCount < maxRetryCount ; retryCount++ {

		waitDuration := getWaitTime(retryCount)
		time.Sleep(waitDuration)

		response, err := httpGetMethod(url)

		if err, isInstance := err.(net.Error); isInstance  { // todo check err
			if err.Timeout() {
				continue
			}
			return nil, err
		} else if shouldRetry(response.StatusCode) {
			continue
		} else {
			return response, err
		}

	}

	return nil, errors.New("Maximum retry count is exceeded.")
}