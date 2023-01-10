package httpclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

// RequestArgs represents  3rd api necessary arguments
//
// Timeout: call timeout, request will cancel after this duration (if no retry)
type RequestArgs struct {
	RequestURL string
	Method     string
	Body       []byte
	Header     map[string]string
	Timeout    time.Duration
}

func (args RequestArgs) validate() error {
	if len(strings.TrimSpace(args.RequestURL)) == 0 {
		return errors.New("httpclient.RequestArgs.validate RequestURL is required")
	}
	if len(strings.TrimSpace(args.Method)) == 0 {
		return errors.New("httpclient.RequestArgs.validate Method is required")
	}

	return nil
}

// Response represents 3rd api response
type Response struct {
	Body []byte
	Err  error
	Code int
}

// Client represent httpclient instance
type Client struct {
	fasthttpClient *fasthttp.Client
}

// NewClient create new Client instance with fasthttp as dependency
func NewClient(options ...func(*fasthttp.Client)) *Client {
	client := &Client{
		fasthttpClient: &fasthttp.Client{
			NoDefaultUserAgentHeader: true,
		},
	}
	for _, option := range options {
		option(client.fasthttpClient)
	}

	return client
}

// Do http request with fasthttp lib
//
// params:
// ctx: context propagation
// args: requirement parameter for execute http request
//
// return:
// response, statusCode, and error (if has)
func (c *Client) Do(ctx context.Context, args RequestArgs) Response {
	var (
		req  = fasthttp.AcquireRequest()
		resp = fasthttp.AcquireResponse()
		err  error
	)

	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()

	// validate argument
	err = args.validate()
	if err != nil {
		return Response{
			Err:  err,
			Code: http.StatusBadRequest,
		}
	}

	// set up request
	req.SetRequestURI(args.RequestURL)
	req.Header.SetMethod(args.Method)

	if args.Body != nil {
		req.SetBody(args.Body)
	}
	if args.Header != nil {
		for k, v := range args.Header {
			req.Header.Add(k, v)
		}
	}

	// Execute http call
	if args.Timeout != 0 {
		err = c.fasthttpClient.DoTimeout(req, resp, args.Timeout)
	} else {
		err = c.fasthttpClient.Do(req, resp)
	}

	if err != nil {
		return Response{
			Err:  fmt.Errorf("httpclient.Do %v", err),
			Code: resp.StatusCode(),
		}
	}

	return Response{
		Body: resp.Body(),
		Code: resp.StatusCode(),
	}
}
