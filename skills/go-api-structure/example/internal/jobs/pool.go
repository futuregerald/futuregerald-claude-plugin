package jobs

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	// ErrQueueFull is returned by TrySubmit when the queue has no room.
	ErrQueueFull = errors.New("jobs: queue full")
	// ErrPoolClosed is returned by both submit methods after Shutdown.
	ErrPoolClosed = errors.New("jobs: pool closed")
)

type Job any

// Handler processes one job. The ctx it receives is the POOL's lifetime
// context, not the context of whoever submitted the job — the submitter has
// usually returned long before the job runs.
type Handler func(ctx context.Context, j Job) error

type Option func(*Pool)

// WithOnError registers a callback for handler errors, panics, and jobs
// dropped by an expired Shutdown deadline. It is called from multiple worker
// goroutines concurrently, so the callback must be safe for concurrent use.
//
// Without it the pool DISCARDS them. A pool cannot know whether an error is
// retryable, fatal or expected, so it must not invent a policy — but it must
// not swallow one silently either, which is what this option exists to
// prevent.
func WithOnError(fn func(j Job, err error)) Option {
	return func(p *Pool) { p.onError = fn }
}

type Pool struct {
	queue   chan Job
	handle  Handler
	onError func(Job, error)

	// The pool's own lifetime context. Storing a context in a struct is
	// normally a red flag — but the prohibition is on storing a REQUEST
	// context, whose deadline then leaks into unrelated work. A long-lived
	// component owning its own lifecycle context is the exception.
	ctx    context.Context
	cancel context.CancelFunc

	closed    chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

// New starts exactly `workers` goroutines. They run until Shutdown.
//
// Panics on workers < 1: a pool with no workers would accept jobs into the
// buffer and never run any of them, which is silent data loss wearing the
// costume of a working queue. Fail at construction instead.
func New(workers, queueSize int, h Handler, opts ...Option) *Pool {
	if workers < 1 {
		panic("jobs: New requires at least one worker")
	}
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		queue:  make(chan Job, queueSize),
		handle: h,
		ctx:    ctx,
		cancel: cancel,
		closed: make(chan struct{}),
	}
	for _, o := range opts {
		o(p)
	}

	// WaitGroup.Go (Go 1.25+) owns both halves of the counter: it increments
	// before starting the goroutine and decrements when that goroutine returns.
	// The Add/Done pair it replaces is the classic place to leak a worker --
	// an Add without its Done makes Wait hang forever, and a Done without its
	// Add panics.
	for i := 0; i < workers; i++ {
		p.wg.Go(p.worker)
	}
	return p
}

func (p *Pool) worker() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case j := <-p.queue:
			// Re-check: cancellation may have landed while this job was being
			// received. Starting work with a context that is already dead
			// wastes effort and muddies the reason it failed.
			if p.ctx.Err() != nil {
				return
			}
			p.run(j)
		case <-p.closed:
			// Shutdown started. Drain whatever is still queued, then exit.
			for {
				select {
				case j := <-p.queue:
					if p.ctx.Err() != nil {
						// Deadline expired mid-drain: dropping the job beats
						// starting it with a context that is already dead.
						return
					}
					p.run(j)
				default:
					return
				}
			}
		}
	}
}

// run isolates one job so a panicking handler kills the job, not the worker.
func (p *Pool) run(j Job) {
	defer func() {
		if r := recover(); r != nil {
			p.report(j, fmt.Errorf("jobs: handler panicked: %v", r))
		}
	}()
	if err := p.handle(p.ctx, j); err != nil {
		p.report(j, err)
	}
}

func (p *Pool) report(j Job, err error) {
	if p.onError != nil {
		p.onError(j, err)
	}
}

// Submit blocks until the queue has room. That blocking IS the backpressure.
// It returns ctx.Err() if the caller gives up first, or ErrPoolClosed after
// Shutdown.
//
// Racing Shutdown, WHICH outcome you get is deliberately not guaranteed: if a
// worker frees a slot at the same moment Shutdown fires, both select cases are
// ready and Go picks one AT RANDOM. So a submission racing shutdown is
// sometimes accepted and sometimes rejected with ErrPoolClosed.
//
// What IS guaranteed: if Submit returns nil, the job runs. That is not free --
// the last worker can exit between the closed-check and the send, so Shutdown
// drains the queue again after the workers are gone (see drainResidual).
// Without that, Submit would return success for work nobody would ever do.
func (p *Pool) Submit(ctx context.Context, j Job) error {
	select {
	case <-p.closed:
		return ErrPoolClosed
	default:
	}

	select {
	case p.queue <- j:
		return nil
	case <-p.closed:
		return ErrPoolClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TrySubmit never blocks: it returns ErrQueueFull instead. Use it where
// shedding load beats waiting — an HTTP handler that would rather answer 503
// than hold a connection open.
func (p *Pool) TrySubmit(j Job) error {
	select {
	case <-p.closed:
		return ErrPoolClosed
	default:
	}

	select {
	case p.queue <- j:
		return nil
	default:
		return ErrQueueFull
	}
}

// Shutdown stops accepting work, waits for queued and in-flight jobs, and
// returns once every worker has exited. If ctx expires first it cancels the
// pool context — aborting in-flight handlers — and still waits for workers to
// return before reporting ctx.Err(). In-flight work is cancelled and awaited,
// never abandoned; abandoning it would leak the very goroutines this method
// exists to reclaim.
//
// Safe to call more than once.
func (p *Pool) Shutdown(ctx context.Context) error {
	p.closeOnce.Do(func() { close(p.closed) })

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		p.cancel()
		<-done
		// Deadline blown: report it. Anything still queued is dropped, and
		// reported so the loss is never silent.
		p.drainResidual(true)
		return ctx.Err()
	}

	// Workers have exited. A Submit that passed its closed-check just before
	// close() may have landed a job in the buffer after the last worker left;
	// without this, that job is accepted (Submit returns nil) and never runs.
	p.drainResidual(false)
	p.cancel()
	return nil
}

// drainResidual empties the queue after the workers are gone. With report=true
// the jobs are abandoned rather than run, and each is passed to OnError so a
// dropped job is always observable.
func (p *Pool) drainResidual(report bool) {
	for {
		select {
		case j := <-p.queue:
			if report {
				p.report(j, fmt.Errorf("jobs: dropped, shutdown deadline expired: %v", j))
				continue
			}
			p.run(j)
		default:
			return
		}
	}
}
