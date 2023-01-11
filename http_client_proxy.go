package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sony/gobreaker"
)

type circuitBreaker struct {
	mapRequestURLToBreaker map[string]*gobreaker.CircuitBreaker
	mutex                  *sync.Mutex
	logger                 logger
	settings               CircuitSetting
}

// CircuitSetting represents settings for gobreaker.CircuitBreaker
type CircuitSetting struct {
	MaxRequests         uint32
	ClosedInterval      time.Duration
	OpenInterval        time.Duration
	ConsecutiveFailures uint32
}

func (b circuitBreaker) getCircuitBreaker(requestURL string) *gobreaker.CircuitBreaker {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if _, ok := b.mapRequestURLToBreaker[requestURL]; !ok {
		// not exist, create
		newCb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:        requestURL,
			MaxRequests: b.settings.MaxRequests,    // MaxRequests pass through cb when state if half-open
			Interval:    b.settings.ClosedInterval, // Reset counter in open
			Timeout:     b.settings.OpenInterval,   // change to half-open when open
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= b.settings.ConsecutiveFailures
			},
			OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
				b.logger.print(fmt.Sprintf("httpclient.circuitbreaker.OnStateChange from %s to %s", from.String(), to.String()))
			},
		})

		b.mapRequestURLToBreaker[requestURL] = newCb
	}

	return b.mapRequestURLToBreaker[requestURL]
}

// ClientBreaker represents httpclient instance with Circuit Breaker proxy
type ClientBreaker struct {
	client *Client
	cb     circuitBreaker
}

// NewClientBreaker ...
func NewClientBreaker(client *Client, circuitSetting CircuitSetting, loggerFunc loggerFunc) *ClientBreaker {
	breaker := &ClientBreaker{
		client: client,
		cb: circuitBreaker{
			mapRequestURLToBreaker: make(map[string]*gobreaker.CircuitBreaker),
			mutex:                  &sync.Mutex{},
			logger:                 loggerFunc,
			settings:               circuitSetting,
		},
	}

	return breaker
}

// Do http request with fasthttp lib and circuit breaker pattern
//
// params:
// ctx: context propagation
// args: requirement parameter for execute http request
//
// return:
// response, statusCode, and error (if has)
func (c *ClientBreaker) Do(ctx context.Context, args RequestArgs) Response {
	cb := c.cb.getCircuitBreaker(args.RequestURL)

	resp, err := cb.Execute(func() (interface{}, error) {
		resp := c.client.Do(ctx, args)
		if resp.Err != nil {
			return resp, resp.Err
		}
		return resp, nil
	})

	// response from client.Do
	if tmp, ok := resp.(Response); ok {
		return tmp
	}

	// response from gobreaker
	return Response{
		Err:  err,
		Code: http.StatusBadRequest,
	}
}
