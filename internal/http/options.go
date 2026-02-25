package http

import "time"

// Options control the default net/http client.
// It is intentionally small (KISS) but enough for production use.
type Options struct {
	Timeout         time.Duration
	KeepAlive       time.Duration
	MaxIdleConns    int
	IdleConnTimeout time.Duration
}

func DefaultOptions() *Options {
	return &Options{
		Timeout:         30 * time.Second,
		KeepAlive:       30 * time.Second,
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
	}
}
