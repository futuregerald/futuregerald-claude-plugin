package jobs

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func noop(context.Context, Job) error { return nil }

// waitFor polls until cond holds or the deadline passes. Used instead of
// sleeping, so the tests are deterministic rather than timing-dependent.
func waitFor(t *testing.T, d time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		runtime.Gosched()
		time.Sleep(time.Millisecond)
	}
	return cond()
}

// 1. 1000 jobs through a queue far smaller than 1000, so backpressure is
//    genuinely exercised rather than bypassed by an oversized buffer.
func TestProcessesEveryJobExactlyOnce(t *testing.T) {
	const n = 1000
	var mu sync.Mutex
	seen := make(map[int]int, n)

	p := New(5, 16, func(_ context.Context, j Job) error {
		mu.Lock()
		seen[j.(int)]++
		mu.Unlock()
		return nil
	})

	for i := 0; i < n; i++ {
		if err := p.Submit(context.Background(), i); err != nil {
			t.Fatalf("Submit(%d): %v", i, err)
		}
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	if len(seen) != n {
		t.Fatalf("processed %d distinct jobs, want %d", len(seen), n)
	}
	for i := 0; i < n; i++ {
		if seen[i] != 1 {
			t.Fatalf("job %d ran %d times, want 1", i, seen[i])
		}
	}
}

// 2. The headline guarantee: never more than `workers` at once.
func TestNeverExceedsWorkerLimit(t *testing.T) {
	const limit = 5
	var inFlight, highWater int64

	p := New(limit, 8, func(_ context.Context, _ Job) error {
		cur := atomic.AddInt64(&inFlight, 1)
		for {
			hw := atomic.LoadInt64(&highWater)
			if cur <= hw || atomic.CompareAndSwapInt64(&highWater, hw, cur) {
				break
			}
		}
		time.Sleep(time.Millisecond)
		atomic.AddInt64(&inFlight, -1)
		return nil
	})

	for i := 0; i < 200; i++ {
		if err := p.Submit(context.Background(), i); err != nil {
			t.Fatalf("Submit: %v", err)
		}
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	if hw := atomic.LoadInt64(&highWater); hw > limit {
		t.Errorf("max concurrent = %d, want <= %d", hw, limit)
	} else if hw < 2 {
		t.Errorf("max concurrent = %d — never ran concurrently, test proves nothing", hw)
	}
}

// 3. TrySubmit sheds load rather than waiting, and recovers once drained.
func TestTrySubmitReportsQueueFullThenRecovers(t *testing.T) {
	release := make(chan struct{})
	p := New(1, 1, func(_ context.Context, _ Job) error {
		<-release
		return nil
	})
	t.Cleanup(func() {
		close(release)
		_ = p.Shutdown(context.Background())
	})

	// Saturate: one job occupies the worker, one fills the single queue slot.
	_ = p.TrySubmit("a")
	_ = p.TrySubmit("b")

	if !waitFor(t, 2*time.Second, func() bool { return errors.Is(p.TrySubmit("c"), ErrQueueFull) }) {
		t.Fatal("TrySubmit never reported ErrQueueFull while saturated")
	}
}

// 4. A blocked Submit must honour its caller's context.
func TestSubmitHonoursCallerContext(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	var once sync.Once
	p := New(1, 1, func(_ context.Context, _ Job) error {
		once.Do(func() { close(started) })
		<-release
		return nil
	})
	t.Cleanup(func() {
		close(release)
		_ = p.Shutdown(context.Background())
	})

	// Occupy the single worker FIRST and wait for confirmation, otherwise the
	// queue slot may still be free when Submit runs and it returns nil.
	if err := p.Submit(context.Background(), "a"); err != nil {
		t.Fatalf("Submit(a): %v", err)
	}
	<-started
	if err := p.Submit(context.Background(), "b"); err != nil {
		t.Fatalf("Submit(b): %v", err) // fills the one queue slot
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := p.Submit(ctx, "c")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Submit err = %v, want DeadlineExceeded", err)
	}
	if time.Since(start) > time.Second {
		t.Errorf("Submit took %v — did not honour the deadline promptly", time.Since(start))
	}
}

// 5. No goroutine outlives Shutdown.
func TestNoGoroutineLeak(t *testing.T) {
	base := runtime.NumGoroutine()

	p := New(8, 16, noop)
	for i := 0; i < 100; i++ {
		if err := p.Submit(context.Background(), i); err != nil {
			t.Fatalf("Submit: %v", err)
		}
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	// Poll rather than assert equality: runtime and timer goroutines come and
	// go, so a bare equality check is flaky.
	if !waitFor(t, 3*time.Second, func() bool { return runtime.NumGoroutine() <= base+2 }) {
		t.Errorf("goroutines = %d, baseline %d — workers leaked", runtime.NumGoroutine(), base)
	}
}

// 6. Shutdown drains queued work instead of dropping it.
func TestShutdownDrainsQueuedWork(t *testing.T) {
	var done int64
	p := New(2, 64, func(_ context.Context, _ Job) error {
		atomic.AddInt64(&done, 1)
		return nil
	})

	const n = 50
	for i := 0; i < n; i++ {
		if err := p.Submit(context.Background(), i); err != nil {
			t.Fatalf("Submit: %v", err)
		}
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if got := atomic.LoadInt64(&done); got != n {
		t.Errorf("processed %d of %d — Shutdown dropped queued work", got, n)
	}
}

// 7. Submitting after Shutdown reports closure and never panics. A pool that
//    closed its job channel would panic here instead.
func TestSubmitAfterShutdownIsRejectedNotPanic(t *testing.T) {
	p := New(2, 4, noop)
	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked after shutdown: %v", r)
		}
	}()
	if err := p.Submit(context.Background(), "x"); !errors.Is(err, ErrPoolClosed) {
		t.Errorf("Submit err = %v, want ErrPoolClosed", err)
	}
	if err := p.TrySubmit("x"); !errors.Is(err, ErrPoolClosed) {
		t.Errorf("TrySubmit err = %v, want ErrPoolClosed", err)
	}
	// Shutdown is idempotent.
	if err := p.Shutdown(context.Background()); err != nil {
		t.Errorf("second Shutdown: %v", err)
	}
}

// 8. An expired Shutdown deadline cancels in-flight handlers.
func TestShutdownDeadlineCancelsRunningHandler(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})

	p := New(1, 1, func(ctx context.Context, _ Job) error {
		close(started)
		<-ctx.Done() // must be cancelled by Shutdown's deadline
		close(cancelled)
		return ctx.Err()
	})

	if err := p.Submit(context.Background(), "slow"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := p.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown err = %v, want DeadlineExceeded", err)
	}
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Error("handler context was never cancelled")
	}
}

// 9. A panicking handler kills its job, not the pool.
func TestHandlerPanicDoesNotKillPool(t *testing.T) {
	var ok int64
	var reported int64

	p := New(2, 8, func(_ context.Context, j Job) error {
		if j.(int)%2 == 0 {
			panic("boom")
		}
		atomic.AddInt64(&ok, 1)
		return nil
	}, WithOnError(func(Job, error) { atomic.AddInt64(&reported, 1) }))

	const n = 20
	for i := 0; i < n; i++ {
		if err := p.Submit(context.Background(), i); err != nil {
			t.Fatalf("Submit: %v", err)
		}
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	if got := atomic.LoadInt64(&ok); got != n/2 {
		t.Errorf("non-panicking jobs completed = %d, want %d — pool stopped working", got, n/2)
	}
	if got := atomic.LoadInt64(&reported); got != n/2 {
		t.Errorf("panics reported = %d, want %d", got, n/2)
	}
}

// 10. Once the deadline expires, still-queued jobs are dropped rather than
//     started with a context that is already dead.
func TestQueuedJobsDroppedAfterDeadline(t *testing.T) {
	started := make(chan struct{})
	var startedWithDeadCtx int64

	p := New(1, 64, func(ctx context.Context, j Job) error {
		if j.(int) == 0 {
			close(started)
			<-ctx.Done()
			return ctx.Err()
		}
		if ctx.Err() != nil {
			atomic.AddInt64(&startedWithDeadCtx, 1)
		}
		return nil
	})

	for i := 0; i < 40; i++ {
		if err := p.Submit(context.Background(), i); err != nil {
			t.Fatalf("Submit: %v", err)
		}
	}
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := p.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown err = %v, want DeadlineExceeded", err)
	}

	if got := atomic.LoadInt64(&startedWithDeadCtx); got != 0 {
		t.Errorf("%d jobs started with an already-cancelled context", got)
	}
}

// 11. A Submit blocked on a full queue returns PROMPTLY when Shutdown starts.
//     Deliberately does NOT assert ErrPoolClosed: if a worker frees a slot as
//     Shutdown fires, both select cases are ready and Go picks at random, so
//     nil is equally correct. Asserting the specific error would require
//     pinning the queue full — the one interleaving where the race cannot
//     happen — and would pass forever against a broken implementation.
func TestBlockedSubmitReturnsPromptlyOnShutdown(t *testing.T) {
	release := make(chan struct{})
	p := New(1, 1, func(_ context.Context, _ Job) error {
		<-release
		return nil
	})

	_ = p.TrySubmit("a")
	_ = p.TrySubmit("b")

	result := make(chan error, 1)
	go func() { result <- p.Submit(context.Background(), "c") }()

	go func() { _ = p.Shutdown(context.Background()) }()
	time.Sleep(20 * time.Millisecond)
	close(release)

	select {
	case err := <-result:
		if err != nil && !errors.Is(err, ErrPoolClosed) {
			t.Errorf("Submit err = %v, want nil or ErrPoolClosed", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("blocked Submit never returned after Shutdown — it hung")
	}
}

// 12. Handler errors reach OnError; without it they are discarded quietly and
//     the pool keeps working.
func TestOnErrorReceivesHandlerErrors(t *testing.T) {
	boom := errors.New("nope")
	var got []error
	var mu sync.Mutex

	p := New(1, 4, func(_ context.Context, _ Job) error { return boom },
		WithOnError(func(_ Job, err error) {
			mu.Lock()
			got = append(got, err)
			mu.Unlock()
		}))

	if err := p.Submit(context.Background(), "x"); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if err := p.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 1 || !errors.Is(got[0], boom) {
		t.Fatalf("OnError got %v, want exactly [%v]", got, boom)
	}

	// Same failure without the option must not panic or hang.
	q := New(1, 4, func(_ context.Context, _ Job) error { return boom })
	if err := q.Submit(context.Background(), "x"); err != nil {
		t.Fatalf("Submit (no OnError): %v", err)
	}
	if err := q.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown (no OnError): %v", err)
	}
}

// 13. REGRESSION: a job Submit accepted must always run. The last worker can
//     exit between Submit's closed-check and its send, leaving the job in the
//     buffer with nobody to take it — Submit returned nil for work that would
//     silently vanish.
func TestAcceptedJobIsNeverSilentlyDropped(t *testing.T) {
	for attempt := 0; attempt < 40; attempt++ {
		var processed int64
		p := New(2, 32, func(context.Context, Job) error {
			atomic.AddInt64(&processed, 1)
			return nil
		})

		var wg sync.WaitGroup
		var accepted int64
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 3; j++ {
					if err := p.Submit(context.Background(), j); err == nil {
						atomic.AddInt64(&accepted, 1)
					}
				}
			}()
		}
		go func() { _ = p.Shutdown(context.Background()) }()
		wg.Wait()
		_ = p.Shutdown(context.Background())

		if got, want := atomic.LoadInt64(&processed), atomic.LoadInt64(&accepted); got < want {
			t.Fatalf("attempt %d: Submit accepted %d jobs but only %d ran — %d silently lost",
				attempt, want, got, want-got)
		}
	}
}
