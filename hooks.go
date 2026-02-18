package yadisk

import (
	"net/http"
	"time"
)

type Hooks struct {
	OnRequest        func(*http.Request)
	OnResponse       func(*http.Response, time.Duration)
	OnRetry          func(RetryEvent)
	OnOperationEvent func(OperationEvent)
}

type RetryEvent struct {
	Attempt     int
	Method      string
	URL         string
	StatusCode  int
	Err         error
	NextBackoff time.Duration
}
