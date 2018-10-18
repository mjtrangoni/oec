package queue

import (
	"github.com/pkg/errors"
	"math"
	"net"
	"net/http"
	"time"
)

const timeout = 5 * time.Second

var DefaultClient = &http.Client{Timeout: timeout} // todo timeout decision

const maxWaitInterval = 5
const maxRetryCount = 5

var retryStatusCodes = map[int]struct{}{
	429: {},
	//408: {},
}

type getMethod func(retryer *Retryer, request *http.Request) (*http.Response, error)

type Retryer struct {
	getMethod getMethod
	request *http.Request
}

func NewRetryer() *Retryer {
	retryer := &Retryer{
		getMethod: getWithExponentialBackoff,
	}
	return retryer
}

func (r *Retryer) Do(request *http.Request) (*http.Response, error) {
	return r.getMethod(r, request)
}

func shouldRetry(statusCode int) bool {
	_, shouldRetry := retryStatusCodes[statusCode]

	if (statusCode >= 500 && statusCode <= 599) || shouldRetry {
		return true
	}
	return false
}

func getWaitTime(retryCount int) time.Duration {
	waitTime := math.Pow(2, float64(retryCount)) * 100
	//waitTime = math.Min(waitTime, float64(maxWaitInterval)) // todo min value
	return time.Duration(waitTime) * time.Millisecond
}

func getWithExponentialBackoff(retryer *Retryer, request *http.Request) (*http.Response, error) {

	for retryCount := 0; retryCount < maxRetryCount; retryCount++ {

		waitDuration := getWaitTime(retryCount)
		time.Sleep(waitDuration)

		response, err := DefaultClient.Do(request)

		if err, isInstance := err.(net.Error); isInstance { // todo check err
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
