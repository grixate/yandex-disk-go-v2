package yadisk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type transport struct {
	httpClient *http.Client
	baseURL    *url.URL
	token      string
	userAgent  string
}

func (c *Client) newRequest(ctx context.Context, method, path string, query url.Values, body io.Reader, contentType string) (*http.Request, error) {
	rel := &url.URL{Path: path}
	if query != nil {
		rel.RawQuery = query.Encode()
	}

	u := c.transport.baseURL.ResolveReference(rel)
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "OAuth "+c.transport.token)
	if c.transport.userAgent != "" {
		req.Header.Set("User-Agent", c.transport.userAgent)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req, nil
}

func isIdempotentMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions:
		return true
	default:
		return false
	}
}

func retryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= 500
}

func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, payload any, out any, expected ...int) (*http.Response, error) {
	var bodyBytes []byte
	contentType := ""
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		bodyBytes = b
		contentType = "application/json"
	}

	attempts := c.retry.MaxRetries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		var body io.Reader
		if bodyBytes != nil {
			body = bytes.NewReader(bodyBytes)
		}

		req, err := c.newRequest(ctx, method, path, query, body, contentType)
		if err != nil {
			return nil, err
		}
		if c.hooks.OnRequest != nil {
			c.hooks.OnRequest(req)
		}

		start := time.Now()
		resp, err := c.transport.httpClient.Do(req)
		duration := time.Since(start)
		if resp != nil && c.hooks.OnResponse != nil {
			c.hooks.OnResponse(resp, duration)
		}

		if err != nil {
			lastErr = err
			if attempt < attempts && isIdempotentMethod(method) {
				backoff := c.backoff(attempt)
				if c.hooks.OnRetry != nil {
					c.hooks.OnRetry(RetryEvent{Attempt: attempt, Method: method, URL: req.URL.String(), Err: err, NextBackoff: backoff})
				}
				if err := sleepWithContext(ctx, backoff); err != nil {
					return nil, err
				}
				continue
			}
			return nil, err
		}

		if retryableStatus(resp.StatusCode) && attempt < attempts && isIdempotentMethod(method) {
			if _, err := io.Copy(io.Discard, resp.Body); err != nil {
				if closeErr := resp.Body.Close(); closeErr != nil {
					return nil, errors.Join(err, closeErr)
				}
				return nil, err
			}
			if err := resp.Body.Close(); err != nil {
				return nil, err
			}
			backoff := c.backoff(attempt)
			if c.hooks.OnRetry != nil {
				c.hooks.OnRetry(RetryEvent{Attempt: attempt, Method: method, URL: req.URL.String(), StatusCode: resp.StatusCode, NextBackoff: backoff})
			}
			if err := sleepWithContext(ctx, backoff); err != nil {
				return nil, err
			}
			continue
		}

		if err := c.decodeResponse(resp, out, expected...); err != nil {
			return resp, err
		}
		return resp, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("request failed after %d attempts", attempts)
}

func (c *Client) doRaw(ctx context.Context, req *http.Request) (*http.Response, error) {
	if c.hooks.OnRequest != nil {
		c.hooks.OnRequest(req)
	}
	start := time.Now()
	resp, err := c.transport.httpClient.Do(req)
	if resp != nil && c.hooks.OnResponse != nil {
		c.hooks.OnResponse(resp, time.Since(start))
	}
	return resp, err
}

func (c *Client) decodeResponse(resp *http.Response, out any, expected ...int) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if closeErr := resp.Body.Close(); closeErr != nil {
			return errors.Join(err, closeErr)
		}
		return err
	}
	closeErr := resp.Body.Close()
	if closeErr != nil {
		return closeErr
	}

	if len(expected) > 0 {
		ok := false
		for _, code := range expected {
			if resp.StatusCode == code {
				ok = true
				break
			}
		}
		if !ok {
			return c.apiErrorFromResponse(resp, body)
		}
	} else if resp.StatusCode >= 400 {
		return c.apiErrorFromResponse(resp, body)
	}

	if out == nil || len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return err
	}
	return nil
}

func (c *Client) apiErrorFromResponse(resp *http.Response, body []byte) error {
	apiErr := &APIError{
		HTTPStatus: resp.StatusCode,
		RequestID:  firstNonEmpty(resp.Header.Get("X-Request-Id"), resp.Header.Get("X-YaRequestId")),
		RawBody:    body,
	}
	_ = json.Unmarshal(body, apiErr)
	if apiErr.Code == "" && apiErr.Message == "" {
		apiErr.Message = strings.TrimSpace(string(body))
	}
	return apiErr
}

func (c *Client) backoff(attempt int) time.Duration {
	backoff := c.retry.BaseDelay
	for i := 1; i < attempt; i++ {
		backoff *= 2
		if backoff >= c.retry.MaxDelay {
			return c.jitter(c.retry.MaxDelay, c.retry.Jitter)
		}
	}
	if backoff > c.retry.MaxDelay {
		backoff = c.retry.MaxDelay
	}
	return c.jitter(backoff, c.retry.Jitter)
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
