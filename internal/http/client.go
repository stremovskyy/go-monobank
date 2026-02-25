package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// HTTPDoer is the minimal interface required from an HTTP client.
// Useful for tests/mocking.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is a thin wrapper over net/http.
// It provides sane defaults and small helpers for JSON APIs.
type Client struct {
	client    HTTPDoer
	transport *http.Transport
	opts      *Options
}

func NewClient(opts *Options) *Client {
	if opts == nil {
		opts = DefaultOptions()
	}

	dialer := &net.Dialer{
		Timeout:   opts.Timeout,
		KeepAlive: opts.KeepAlive,
	}

	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, addr)
		},
		ForceAttemptHTTP2: true,
		MaxIdleConns:      opts.MaxIdleConns,
		IdleConnTimeout:   opts.IdleConnTimeout,
	}

	hc := &http.Client{
		Timeout: opts.Timeout,
		Transport: tr,
	}

	return &Client{client: hc, transport: tr, opts: opts}
}

// SetClient overrides underlying net/http client.
// If provided client has Timeout set, it will be respected for request context.
func (c *Client) SetClient(hc *http.Client) {
	if hc == nil {
		return
	}
	c.client = hc
}

func (c *Client) Do(req *http.Request) (*http.Response, []byte, error) {
	if req == nil {
		return nil, nil, fmt.Errorf("http: request is nil")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, err
	}
	return resp, body, nil
}

// NewJSONRequest creates an http.Request with JSON body (if payload != nil)
// and sets Content-Type/Accept to application/json.
func NewJSONRequest(ctx context.Context, method, url string, payload any) (*http.Request, error) {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("json marshal: %w", err)
		}
		body = bytes.NewReader(b)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// WithTimeout returns a new context with timeout based on duration.
// If d <= 0, returns the original ctx.
func WithTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if d <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}
