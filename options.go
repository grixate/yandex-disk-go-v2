package yadisk

import (
	"errors"
	"net/http"
	"net/url"
	"time"
)

const defaultBaseURL = "https://cloud-api.yandex.net/v1"

var (
	errEmptyToken = errors.New("oauth token is required")
)

type Option func(*config) error

type config struct {
	token       string
	baseURL     *url.URL
	httpClient  *http.Client
	userAgent   string
	retryPolicy RetryPolicy
	hooks       Hooks
	worker      WorkerConfig
}

type RetryPolicy struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Jitter     float64
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   2 * time.Second,
		Jitter:     0.25,
	}
}

type WorkerConfig struct {
	PollInterval time.Duration
	MaxInterval  time.Duration
	Jitter       float64
	QueueSize    int
}

func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		PollInterval: 500 * time.Millisecond,
		MaxInterval:  10 * time.Second,
		Jitter:       0.15,
		QueueSize:    256,
	}
}

func defaultConfig() (*config, error) {
	u, err := url.Parse(defaultBaseURL)
	if err != nil {
		return nil, err
	}

	return &config{
		baseURL:     u,
		httpClient:  http.DefaultClient,
		retryPolicy: DefaultRetryPolicy(),
		worker:      DefaultWorkerConfig(),
	}, nil
}

func WithOAuthToken(token string) Option {
	return func(c *config) error {
		c.token = token
		return nil
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *config) error {
		if client == nil {
			return errors.New("http client must not be nil")
		}
		c.httpClient = client
		return nil
	}
}

func WithBaseURL(rawURL string) Option {
	return func(c *config) error {
		u, err := url.Parse(rawURL)
		if err != nil {
			return err
		}
		c.baseURL = u
		return nil
	}
}

func WithUserAgent(userAgent string) Option {
	return func(c *config) error {
		c.userAgent = userAgent
		return nil
	}
}

func WithRetryPolicy(policy RetryPolicy) Option {
	return func(c *config) error {
		if policy.BaseDelay <= 0 || policy.MaxDelay <= 0 {
			return errors.New("retry delays must be positive")
		}
		if policy.MaxRetries < 0 {
			return errors.New("max retries must be non-negative")
		}
		if policy.Jitter < 0 {
			return errors.New("retry jitter must be >= 0")
		}
		c.retryPolicy = policy
		return nil
	}
}

func WithHooks(hooks Hooks) Option {
	return func(c *config) error {
		c.hooks = hooks
		return nil
	}
}

func WithWorkerConfig(cfg WorkerConfig) Option {
	return func(c *config) error {
		if cfg.PollInterval <= 0 || cfg.MaxInterval <= 0 {
			return errors.New("worker intervals must be positive")
		}
		if cfg.QueueSize <= 0 {
			return errors.New("worker queue size must be positive")
		}
		if cfg.Jitter < 0 {
			return errors.New("worker jitter must be >= 0")
		}
		c.worker = cfg
		return nil
	}
}
