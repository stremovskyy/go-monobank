package go_monobank

import (
	"net/http"
	"strings"
	"time"

	"github.com/stremovskyy/go-monobank/consts"
	internalhttp "github.com/stremovskyy/go-monobank/internal/http"
)

type clientConfig struct {
	baseURL string

	httpOptions *internalhttp.Options
	httpClient  *http.Client

	// defaultToken is used when request.Merchant.Token is empty.
	defaultToken string

	// webhookPublicKeyBase64 is base64-encoded PEM returned by /api/merchant/pubkey.
	webhookPublicKeyBase64 string

	// webhookPublicKeyPEM is raw PEM (decoded).
	webhookPublicKeyPEM []byte
}

func defaultClientConfig() *clientConfig {
	return &clientConfig{
		baseURL:     consts.DefaultBaseURL,
		httpOptions: internalhttp.DefaultOptions(),
	}
}

// Option configures Monobank client.
type Option func(*clientConfig)

// WithBaseURL overrides API base URL.
func WithBaseURL(baseURL string) Option {
	return func(c *clientConfig) {
		baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
		if baseURL != "" {
			c.baseURL = baseURL
		}
	}
}

func WithTimeout(d time.Duration) Option {
	return func(c *clientConfig) {
		c.httpOptions.Timeout = d
	}
}

func WithKeepAlive(d time.Duration) Option {
	return func(c *clientConfig) {
		c.httpOptions.KeepAlive = d
	}
}

func WithMaxIdleConns(n int) Option {
	return func(c *clientConfig) {
		c.httpOptions.MaxIdleConns = n
	}
}

func WithIdleConnTimeout(d time.Duration) Option {
	return func(c *clientConfig) {
		c.httpOptions.IdleConnTimeout = d
	}
}

// WithClient overrides underlying net/http client.
func WithClient(cl *http.Client) Option {
	return func(c *clientConfig) {
		c.httpClient = cl
		if cl != nil {
			c.httpOptions.Timeout = cl.Timeout
		}
	}
}

// WithToken sets default X-Token.
// If request.Merchant.Token is empty, client will use this token.
func WithToken(token string) Option {
	return func(c *clientConfig) {
		c.defaultToken = strings.TrimSpace(token)
	}
}

// WithWebhookPublicKeyBase64 sets base64-encoded PEM public key (from /api/merchant/pubkey).
func WithWebhookPublicKeyBase64(key string) Option {
	return func(c *clientConfig) {
		c.webhookPublicKeyBase64 = strings.TrimSpace(key)
	}
}

// WithWebhookPublicKeyPEM sets raw PEM public key.
func WithWebhookPublicKeyPEM(pemBytes []byte) Option {
	return func(c *clientConfig) {
		if len(pemBytes) == 0 {
			return
		}
		c.webhookPublicKeyPEM = append([]byte(nil), pemBytes...)
	}
}

// NewClient creates Monobank client with custom options.
func NewClient(opts ...Option) Monobank {
	cfg := defaultClientConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	hc := internalhttp.NewClient(cfg.httpOptions)
	if cfg.httpClient != nil {
		hc.SetClient(cfg.httpClient)
	}

	return &client{
		http: hc,
		cfg:  cfg,
	}
}

// NewDefaultClient returns client with defaults.
func NewDefaultClient() Monobank { return NewClient() }
