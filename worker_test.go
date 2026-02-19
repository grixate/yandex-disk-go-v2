package yadisk

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestOperationWorkerLifecycle(t *testing.T) {
	var polls int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/disk/operations/op-1" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		n := atomic.AddInt32(&polls, 1)
		if n < 2 {
			if _, err := fmt.Fprint(w, `{"status":"in-progress"}`); err != nil {
				t.Fatalf("write response: %v", err)
			}
			return
		}
		if _, err := fmt.Fprint(w, `{"status":"success"}`); err != nil {
			t.Fatalf("write response: %v", err)
		}
	})

	client.workerCfg.PollInterval = 20 * time.Millisecond
	client.workerCfg.MaxInterval = 60 * time.Millisecond
	client.Worker = newOperationWorker(client, client.workerCfg)

	ctx := context.Background()
	if err := client.Worker.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}

	events := make(chan OperationEvent, 4)
	if err := client.Worker.Watch(OperationRef{ID: "op-1"}, func(e OperationEvent) {
		events <- e
	}); err != nil {
		t.Fatalf("watch: %v", err)
	}

	timeout := time.After(2 * time.Second)
	for {
		select {
		case e := <-events:
			if e.Done {
				if e.Status != "success" {
					t.Fatalf("status=%s", e.Status)
				}
				if err := client.Worker.Stop(context.Background()); err != nil {
					t.Fatalf("stop: %v", err)
				}
				return
			}
		case <-timeout:
			t.Fatal("timeout waiting for terminal event")
		}
	}
}

func TestOperationWorkerStopContext(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := fmt.Fprint(w, `{"status":"in-progress"}`); err != nil {
			t.Fatalf("write response: %v", err)
		}
	})

	if err := client.Worker.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Worker.Stop(ctx); err != nil {
		t.Fatalf("stop: %v", err)
	}
}
