package yadisk

import (
	"context"
	"errors"
	"sync"
	"time"
)

type OperationEvent struct {
	Ref    OperationRef
	Status string
	Done   bool
	Err    error
}

type watchState struct {
	ref      OperationRef
	handlers []func(OperationEvent)
	interval time.Duration
	nextPoll time.Time
}

type OperationWorker struct {
	client *Client
	cfg    WorkerConfig

	mu       sync.Mutex
	watchers map[string]*watchState
	started  bool
	cancel   context.CancelFunc
	done     chan struct{}
}

func newOperationWorker(client *Client, cfg WorkerConfig) *OperationWorker {
	return &OperationWorker{
		client:   client,
		cfg:      cfg,
		watchers: make(map[string]*watchState),
	}
}

func (w *OperationWorker) Start(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.started {
		return nil
	}
	loopCtx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.done = make(chan struct{})
	w.started = true
	go w.loop(loopCtx)
	return nil
}

func (w *OperationWorker) Stop(ctx context.Context) error {
	w.mu.Lock()
	if !w.started {
		w.mu.Unlock()
		return nil
	}
	cancel := w.cancel
	done := w.done
	w.mu.Unlock()

	cancel()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
	}

	w.mu.Lock()
	w.started = false
	w.cancel = nil
	w.done = nil
	w.mu.Unlock()
	return nil
}

func (w *OperationWorker) Watch(ref OperationRef, handler func(OperationEvent)) error {
	if handler == nil {
		return errors.New("handler is required")
	}
	if ref.ID == "" && ref.Href != "" {
		if parsed := operationRefFromLink(&Link{Href: ref.Href}); parsed != nil {
			ref.ID = parsed.ID
			ref.Href = parsed.Href
		}
	}
	if ref.ID == "" {
		return errors.New("operation id is required")
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	state, ok := w.watchers[ref.ID]
	if !ok {
		state = &watchState{
			ref:      ref,
			interval: w.cfg.PollInterval,
			nextPoll: time.Now(),
		}
		w.watchers[ref.ID] = state
	}
	state.handlers = append(state.handlers, handler)
	return nil
}

func (w *OperationWorker) loop(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	defer close(w.done)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *OperationWorker) tick(ctx context.Context) {
	now := time.Now()

	w.mu.Lock()
	states := make([]*watchState, 0, len(w.watchers))
	for _, state := range w.watchers {
		if !state.nextPoll.After(now) {
			states = append(states, state)
		}
	}
	w.mu.Unlock()

	for _, state := range states {
		status, err := w.client.Operations.GetStatus(ctx, OperationStatusRequest{OperationID: state.ref.ID})
		event := OperationEvent{Ref: state.ref}
		if err != nil {
			event.Err = err
			w.bump(state.ref.ID, true)
			w.dispatch(state.handlers, event)
			continue
		}

		event.Status = status.Status
		event.Done = status.IsTerminal()
		w.dispatch(state.handlers, event)

		if event.Done {
			w.remove(state.ref.ID)
			continue
		}
		w.bump(state.ref.ID, false)
	}
}

func (w *OperationWorker) bump(id string, onError bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	state, ok := w.watchers[id]
	if !ok {
		return
	}
	next := state.interval
	if onError {
		next *= 2
		if next > w.cfg.MaxInterval {
			next = w.cfg.MaxInterval
		}
	}
	state.interval = next
	state.nextPoll = time.Now().Add(w.client.jitter(next, w.cfg.Jitter))
}

func (w *OperationWorker) remove(id string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.watchers, id)
}

func (w *OperationWorker) dispatch(handlers []func(OperationEvent), event OperationEvent) {
	if w.client.hooks.OnOperationEvent != nil {
		w.client.hooks.OnOperationEvent(event)
	}
	for _, h := range handlers {
		hCopy := h
		eventCopy := event
		go hCopy(eventCopy)
	}
}
