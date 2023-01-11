package httpclient

import (
	"fmt"
	"net"
	"time"

	"github.com/valyala/fasthttp"
)

// WithMaxConnsPerHost set maximum number of connections per each host which may be established
func WithMaxConnsPerHost(maxConnsPerHost int) func(c *fasthttp.Client) {
	return func(c *fasthttp.Client) {
		c.MaxConnsPerHost = maxConnsPerHost
	}
}

// WithIdleKeepAliveDuration set idle time for Keep-Alive connection
// Time reconnect = min(MaxIdleConnDuration,MaxConnDuration)
func WithIdleKeepAliveDuration(duration time.Duration) func(*fasthttp.Client) {
	return func(c *fasthttp.Client) {
		c.MaxIdleConnDuration = duration
	}
}

// WithMaxIdemponentCallAttempts set retry time when call api
// Default = 5
func WithMaxIdemponentCallAttempts(idemponent int) func(*fasthttp.Client) {
	return func(c *fasthttp.Client) {
		c.MaxIdemponentCallAttempts = idemponent
	}
}

// WithDialTimeout tcp Dial with duration timeout
func WithDialTimeout(duration time.Duration) func(*fasthttp.Client) {
	return func(c *fasthttp.Client) {
		c.Dial = func(addr string) (net.Conn, error) {
			// TODO optimize tcp dial
			conn, err := fasthttp.DialTimeout(addr, duration)
			if err != nil {
				return nil, fmt.Errorf("httpclient.dial %v", err)
			}

			return conn, nil
		}
	}
}
