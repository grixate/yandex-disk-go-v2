package yadisk

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"
)

type Client struct {
	transport  *transport
	retry      RetryPolicy
	hooks      Hooks
	workerCfg  WorkerConfig
	randSource *rand.Rand
	randMu     sync.Mutex

	Disk       *DiskService
	Resources  *ResourcesService
	Uploads    *UploadsService
	Public     *PublicService
	Trash      *TrashService
	Operations *OperationsService
	Worker     *OperationWorker
}

func NewClient(opts ...Option) (*Client, error) {
	cfg, err := defaultConfig()
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}
	if cfg.token == "" {
		return nil, errEmptyToken
	}
	if cfg.httpClient == nil {
		return nil, errors.New("http client is required")
	}

	c := &Client{
		transport: &transport{
			httpClient: cfg.httpClient,
			baseURL:    cfg.baseURL,
			token:      cfg.token,
			userAgent:  cfg.userAgent,
		},
		retry:      cfg.retryPolicy,
		hooks:      cfg.hooks,
		workerCfg:  cfg.worker,
		randSource: rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	c.Disk = &DiskService{client: c}
	c.Resources = &ResourcesService{client: c}
	c.Uploads = &UploadsService{client: c}
	c.Public = &PublicService{client: c}
	c.Trash = &TrashService{client: c}
	c.Operations = &OperationsService{client: c}
	c.Worker = newOperationWorker(c, cfg.worker)
	return c, nil
}

func (c *Client) Close(ctx context.Context) error {
	if c.Worker == nil {
		return nil
	}
	return c.Worker.Stop(ctx)
}

func (c *Client) jitter(duration time.Duration, jitter float64) time.Duration {
	if jitter <= 0 {
		return duration
	}

	c.randMu.Lock()
	delta := (c.randSource.Float64()*2 - 1) * jitter
	c.randMu.Unlock()

	jittered := float64(duration) * (1 + delta)
	if jittered < 0 {
		return 0
	}
	return time.Duration(jittered)
}
