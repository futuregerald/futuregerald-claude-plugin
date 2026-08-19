# Concurrency: Goroutines, Channels, and Cleaning Them Up

Blocks marked **complete file** compile as written. Blocks marked **fragment** omit imports or
surrounding code, or are deliberately buggy examples, and are illustrative only.

- [The problem this solves](#the-problem-this-solves)
- [Where a pool lives](#where-a-pool-lives)
- [The pool](#the-pool)
- [Using it](#using-it)
- [Why each piece is there](#why-each-piece-is-there)
- [The simpler alternative: errgroup](#the-simpler-alternative-errgroup)
- [Classic bugs](#classic-bugs)

## The problem this solves

You have 1,000 things to do and you cannot do them all at once. Spawning
`go doWork(item)` in a loop starts 1,000 goroutines immediately: 1,000 simultaneous database
connections, 1,000 outbound HTTP calls, and a memory spike that scales with input you do not
control. Doing them one at a time is safe but slow.

What you want is a **bounded worker pool**: a fixed number of goroutines (say 5) pulling work
from a queue until the queue is empty. Concurrency stays capped no matter how much work
arrives.

Two terms used throughout:

- **Backpressure** — what happens when work arrives faster than it is processed. Either the
  producer waits (blocks) or the work is rejected. Something must give; the only wrong answer
  is an unbounded queue that grows until the process dies.
- **Goroutine leak** — a goroutine that never returns. It holds its stack and whatever it
  references forever. Leaks are invisible until the process runs out of memory, which is why
  the tests below assert goroutine count returns to baseline.

## Where a pool lives

`internal/jobs` — a **capability package**: in-process machinery, named for what it does.
(Adapters are also often named for a capability, to dodge library name collisions — the
difference is not the name but whether the code leaves the process. This one does not.)
It is not an adapter (nothing leaves the process)
and not domain logic (it encodes no business rule). It is a reusable mechanism, so it sits
beside them and is wired in `main` like everything else.

If a domain service needs to enqueue work, it declares its own narrow interface and `main`
passes the pool in — the same consumer-declares-what-it-needs rule used everywhere else in
this skill.

## The pool

### `internal/jobs/pool.go` (complete file)

```go
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
```

## Using it

```go
// fragment — 5 workers, a queue of 16, and 1000 jobs pushed through it
pool := jobs.New(5, 16, func(ctx context.Context, j jobs.Job) error {
	return sendWelcomeEmail(ctx, j.(string))
}, jobs.WithOnError(func(j jobs.Job, err error) {
	log.Error("job failed", "job", j, "err", err)
}))

for i := 0; i < 1000; i++ {
	// Blocks whenever 16 are already queued — that is backpressure working.
	if err := pool.Submit(ctx, fmt.Sprintf("user-%d", i)); err != nil {
		return err
	}
}

shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
return pool.Shutdown(shutdownCtx)
```

The queue holds 16 while 1,000 jobs flow through it. At most 5 run at once, no matter how fast
the loop pushes.

## Why each piece is there

| Piece | What breaks without it |
|---|---|
| `sync.WaitGroup` | `Shutdown` returns while workers are still running; the process exits mid-job |
| `sync.Once` on close | A second `Shutdown` closes an already-closed channel and panics |
| **Never closing `queue`** | A `Submit` blocked on send would panic — sending on a closed channel is a panic, and you cannot check "is it closed" first without a race |
| `drainResidual` after `wg.Wait` | A job accepted in the instant the last worker exits is queued and never run — `Submit` returned `nil` for work that silently vanishes |
| Panic in `New` on `workers < 1` | A zero-worker pool accepts everything and runs nothing |
| `recover` in `run` | One panicking job kills a worker permanently; after N panics the pool silently stops working |
| Pool ctx (not request ctx) | Jobs get cancelled the instant the HTTP request that queued them returns |
| Buffered `queue` | An unbuffered channel makes `Submit` wait for a free worker rather than a free slot — usually fine, but you lose the queue |

**The channel is never closed.** This is the design decision people get wrong most often.
Closing a channel is the natural way to say "no more work", but a sender blocked in
`Submit` would then panic, and Go gives you no safe way to check for closure before sending.
Signalling shutdown on a *separate* `closed` channel keeps senders safe: they select on it and
return `ErrPoolClosed` instead of panicking.

## The simpler alternative: errgroup

For work scoped to a single request, a pool is usually more machinery than you need.
`errgroup` with a limit does bounded concurrency in four lines and waits for the results:

```go
// fragment — fetch 100 URLs, at most 10 at a time, inside one request
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(10)

for _, u := range urls {
	u := u // not needed on Go 1.22+, where loop vars are per-iteration
	g.Go(func() error {
		return fetch(ctx, u)
	})
}
return g.Wait() // returns the first error and cancels the rest
```

**Choosing between them:**

| Use | When |
|---|---|
| `errgroup.SetLimit` | The caller waits for the results, within one request. Errors propagate to the caller; cancellation is automatic. |
| A `Pool` | The work outlives the request, or has its own lifecycle: background jobs, queue consumers, anything that must survive the response and be drained on shutdown. |

If you are reaching for a pool inside a single request handler, you probably want `errgroup`.

## Testing concurrent code

Concurrency tests have a problem the code under test does not: there is no way to ask the
runtime whether it has finished being busy. So they poll (`waitFor` in `pool_test.go`) or they
sleep, and both answer "has it happened yet?" with a guess. A generous timeout makes the suite
slow; a tight one makes it flaky on a loaded CI box. Neither is an assertion.

### `testing/synctest` — virtual time and a settled state

**Go 1.25+.** It shipped experimentally in 1.24 behind `GOEXPERIMENT=synctest` with a different
entry point (`synctest.Run(func())`); the 1.25 API below is the one that stays.

`synctest.Test` runs a function inside a *bubble*, along with every goroutine started in it.
Two things change inside:

- **The clock is virtual.** It advances only when every goroutine in the bubble is durably
  blocked, and then jumps straight to the next time something wakes. Simulated seconds cost no
  wall-clock time.
- **`synctest.Wait` blocks until every other goroutine in the bubble is durably blocked** —
  a state where nothing is merely "not done yet". Assert after `Wait` and you are asserting
  about a settled system rather than photographing a race.

`internal/jobs/pool_test.go` uses it to measure backpressure exactly, which no amount of
polling can do — polling can see that submissions stopped, not that they stopped at the right
number:

```go
// fragment — see TestSubmitBlocksWhenPoolIsSaturated for the whole test
synctest.Test(t, func(t *testing.T) {
	p := New(2, 3, func(context.Context, Job) error {
		time.Sleep(time.Second) // virtual: instant
		return nil
	})

	var accepted atomic.Int64
	go func() {
		for i := range 12 {
			if err := p.Submit(context.Background(), i); err != nil {
				t.Errorf("Submit(%d): %v", i, err)
				return
			}
			accepted.Add(1)
		}
	}()

	// Both workers are asleep in a handler and the submitter is parked on a
	// full queue. 2 in flight + 3 queued, and the 6th Submit is blocked.
	synctest.Wait()
	if got := accepted.Load(); got != 5 {
		t.Fatalf("accepted %d before blocking, want 5 — the queue is not bounded", got)
	}

	// One second of virtual time frees exactly one slot per worker.
	time.Sleep(time.Second)
	synctest.Wait()
	if got := accepted.Load(); got != 7 {
		t.Fatalf("accepted %d, want 7", got)
	}
})
```

**"Durably blocked" is narrower than "blocked", and this is the whole gotcha.** Blocked in a
way only another goroutine *in the bubble* can undo:

| Durably blocking | Not durably blocking |
|---|---|
| Send/receive on a channel created in the bubble | Locking a `sync.Mutex` or `RWMutex` |
| A `select` where every case is such a channel | Network or file I/O |
| `time.Sleep` | Syscalls |
| `sync.Cond.Wait` | Anything waiting on a goroutine outside the bubble |
| `sync.WaitGroup.Wait`, when `Add` ran in the bubble | |

A goroutine sitting on something in the right-hand column never lets the bubble go idle, so
time stops advancing and `Wait` never returns. `synctest.Test` reports that as a deadlock panic
rather than hanging, which is the failure mode you want: bubble a component that touches the
network and the test tells you immediately instead of at the CI timeout.

### `sync.WaitGroup.Go`

**Go 1.25+.** `wg.Go(f)` increments the counter, runs `f` in a new goroutine, and decrements
when it returns. `New` uses it to start the workers:

```go
// fragment — what New used to do
p.wg.Add(workers)
for i := 0; i < workers; i++ {
	go p.worker() // and `defer p.wg.Done()` at the top of worker
}

// fragment — what it does now
for i := 0; i < workers; i++ {
	p.wg.Go(p.worker)
}
```

The counter can no longer get out of step with the goroutines, which makes the `Add`-inside-
the-goroutine bug below unrepresentable. **When converting existing code, delete the
`defer wg.Done()` from the function body.** `Go` owns the decrement; a leftover `Done`
double-decrements and panics with `sync: negative WaitGroup counter` — on the first shutdown,
not at the call site that caused it.

### `goleak` for leak assertions

`TestNoGoroutineLeak` in `pool_test.go` compares `runtime.NumGoroutine()` against a baseline
and tolerates a couple of stragglers, because runtime and timer goroutines come and go. That is
the weakest test in the file: it counts goroutines, so it can tell you one leaked but never
which one, and the tolerance that stops it flaking is also what lets a single leak through.

[`go.uber.org/goleak`](https://github.com/uber-go/goleak) does the job properly — it diffs the
goroutine profile and fails with the *stack* of whatever is still running:

```go
// fragment — one test
func TestShutdownLeavesNothingRunning(t *testing.T) {
	defer goleak.VerifyNone(t)
	// ...exercise and shut down the component...
}

// fragment — or the whole package at once
func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }
```

This module deliberately does not take that dependency: it is a teaching example, and every
`require` line is something a reader has to evaluate before copying anything. In a real service
it is worth taking — a leak that names the goroutine is a five-minute fix, and a leak that only
shows up as a rising count is an afternoon.

Note that `synctest` covers part of the same ground for free: `synctest.Test` does not return
until every goroutine in the bubble has exited, so a worker that outlives its shutdown fails
the test as a deadlock.

### The missing piece: HTTP inside a bubble

Testing an HTTP client or server under virtual time does not work yet. `httptest.NewServer`
listens on real loopback, and network I/O is not durably blocking — the bubble never goes idle,
so the clock never moves. **`httptest.NewTestServer`, which serves over an in-memory connection
instead, lands in Go 1.27.** This module targets `go 1.25.0`, so it is not used here.

Until you are on 1.27 the options are `net.Pipe` for a hand-built connection (this is what the
`testing/synctest` package documentation's own HTTP example does), or skipping the transport
entirely and calling the handler with `httptest.NewRecorder` — which is what every test in
`internal/httpapi` does anyway, and which is faster and clearer regardless of Go version.

## Classic bugs

Each block below is deliberately wrong.

**Leaking a goroutine on an unbuffered channel.**

```go
// fragment — BROKEN. If nobody ever receives, this goroutine never exits.
results := make(chan int) // unbuffered
go func() {
	results <- expensive() // blocks forever if the reader gave up
}()

if time.Now().After(deadline) {
	return errTimeout // reader leaves; the goroutine above leaks
}
```

Fix: buffer the channel (`make(chan int, 1)`) so the send always completes, or select on
`ctx.Done()` in the sender.

**`WaitGroup.Add` inside the goroutine.**

```go
// fragment — BROKEN. Add races with Wait; Wait may return before anything starts.
for _, item := range items {
	go func(i Item) {
		wg.Add(1) // too late
		defer wg.Done()
		process(i)
	}(item)
}
wg.Wait()
```

Fix: `wg.Add(1)` before `go`, or `wg.Add(len(items))` once up front.

**Ranging a channel nobody closes.**

```go
// fragment — BROKEN. range blocks forever once the channel drains.
for v := range ch { // no sender ever calls close(ch)
	use(v)
}
```

Fix: exactly one owner — the sender — closes it, and only when no further sends will happen.
Never close from the receiving side, and never from multiple senders.

**Fire-and-forget with the request context.**

```go
// fragment — BROKEN. ctx is cancelled the moment the handler returns.
go func() {
	sendReceipt(ctx, order) // almost always cancelled before it finishes
}()
```

Fix: `context.WithoutCancel(ctx)` for a one-off (keeps trace IDs, drops the deadline), or
submit it to a pool if it happens often enough to need bounding.

**One that stopped being a bug.** Before Go 1.22, loop variables were shared across
iterations, so every goroutine below saw the final value:

```go
// fragment — a real bug before Go 1.22; correct on 1.22+
for _, item := range items {
	go func() { process(item) }() // pre-1.22: all goroutines saw the last item
}
```

Go 1.22 made loop variables per-iteration, so this is now correct. On an older toolchain the
fix is the classic `item := item` shadow. Check your `go.mod` before trusting it.
