package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sony/gobreaker"
)

type breaker struct {
	mapRequestURLToBreaker map[string]*gobreaker.CircuitBreaker
	mutex                  *sync.Mutex
	logger                 logger
}

func (b breaker) getCircuitBreaker(requestURL string) *gobreaker.CircuitBreaker {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if _, ok := b.mapRequestURLToBreaker[requestURL]; !ok {
		// not exist, create
		newCb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:        requestURL,
			MaxRequests: 5,                // MaxRequests pass through cb when state if half-open
			Interval:    time.Minute,      // Reset counter in open
			Timeout:     time.Second * 10, // change to half-open when open
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 10
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
	client  *Client
	breaker breaker
}

// NewClientBreaker ...
func NewClientBreaker(client *Client, loggerFunc loggerFunc) *ClientBreaker {
	breaker := &ClientBreaker{
		client: client,
		breaker: breaker{
			mapRequestURLToBreaker: make(map[string]*gobreaker.CircuitBreaker),
			mutex:                  &sync.Mutex{},
			logger:                 loggerFunc,
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
	cb := c.breaker.getCircuitBreaker(args.RequestURL)

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
