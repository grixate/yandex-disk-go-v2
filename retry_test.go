package yadisk

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryOnGet(t *testing.T) {
	var attempts int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		w.Header().Set("Content-Type", "application/json")
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"InternalError"}`)
			return
		}
		fmt.Fprint(w, `{}`)
	})

	client.retry = RetryPolicy{MaxRetries: 4, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond, Jitter: 0}
	_, err := client.Disk.Get(context.Background(), DiskGetRequest{})
	if err != nil {
		t.Fatalf("disk get err: %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("attempts = %d want 3", got)
	}
}

func TestNoRetryOnPost(t *testing.T) {
	var attempts int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"InternalError"}`)
	})

	client.retry = RetryPolicy{MaxRetries: 4, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond, Jitter: 0}
	_, err := client.Resources.Copy(context.Background(), CopyMoveRequest{From: "disk:/a", Path: "disk:/b"})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("attempts = %d want 1", got)
	}
}
