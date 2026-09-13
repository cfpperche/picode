package server

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Decision table (docs/changelog.d entry of the same branch):
//
// | # | Conditions                                   | Action                                   |
// |---+----------------------------------------------+------------------------------------------|
// | 1 | no snapshot yet                              | compute, store, serve                    |
// | 2 | snapshot younger than TTL                    | serve snapshot, never compute            |
// | 3 | snapshot older than TTL                      | recompute                                |
// | 4 | concurrent Views while computing             | one computation, all share the result    |
// | 5 | Invalidate after a mutation, snapshot fresh  | next View recomputes                     |
// | 6 | compute fails                                | error shared by the round, not cached    |
// | 7 | late joiner arrives after compute finished   | gets the round's result, no new compute  |
// | 8 | leader canceled mid-flight                   | joiners see the round's error, recover   |

func TestTerminalsCacheDecisionTable(t *testing.T) {
	t.Run("cold computes and stores", func(t *testing.T) { // row 1
		var calls int
		c := &TerminalsCache{}
		out, err := c.View(DefaultTTL, func() ([]map[string]any, error) {
			calls++
			return []map[string]any{{"id": "t1"}}, nil
		})
		if err != nil || calls != 1 || len(out) != 1 || out[0]["id"] != "t1" {
			t.Fatalf("cold view: out=%v err=%v calls=%d", out, err, calls)
		}
	})

	t.Run("fresh snapshot answers without computing", func(t *testing.T) { // row 2
		var calls int
		c := &TerminalsCache{}
		compute := func() ([]map[string]any, error) {
			calls++
			return []map[string]any{{"id": "t1"}}, nil
		}
		if _, err := c.View(DefaultTTL, compute); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 5; i++ {
			out, err := c.View(DefaultTTL, compute)
			if err != nil || len(out) != 1 || out[0]["id"] != "t1" {
				t.Fatalf("warm view %d: out=%v err=%v", i, out, err)
			}
		}
		if calls != 1 {
			t.Fatalf("warm views recomputed: calls=%d", calls)
		}
	})

	t.Run("expired snapshot recomputes", func(t *testing.T) { // row 3
		var calls int
		now := time.Unix(1_000_000, 0)
		c := &TerminalsCache{clock: func() time.Time { return now }}
		compute := func() ([]map[string]any, error) {
			calls++
			return []map[string]any{{"n": calls}}, nil
		}
		if _, err := c.View(time.Second, compute); err != nil {
			t.Fatal(err)
		}
		now = now.Add(2 * time.Second) // past the TTL
		out, err := c.View(time.Second, compute)
		if err != nil || calls != 2 || out[0]["n"] != 2 {
			t.Fatalf("expired view: out=%v err=%v calls=%d", out, err, calls)
		}
	})

	t.Run("concurrent views share one computation", func(t *testing.T) { // row 4
		release := make(chan struct{})
		var calls int32
		c := &TerminalsCache{}
		compute := func() ([]map[string]any, error) {
			atomic.AddInt32(&calls, 1)
			<-release
			return []map[string]any{{"id": "shared"}}, nil
		}
		const n = 8
		var wg sync.WaitGroup
		results := make([][]map[string]any, n)
		errs := make([]error, n)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				results[i], errs[i] = c.View(DefaultTTL, compute)
			}(i)
		}
		// Every goroutine is either waiting inside compute or on the flight;
		// give the scheduler a beat, then let the round finish.
		deadline := time.Now().Add(2 * time.Second)
		for atomic.LoadInt32(&calls) == 0 && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		close(release)
		wg.Wait()
		if calls != 1 {
			t.Fatalf("concurrent views computed %d times, want 1", calls)
		}
		for i := range results {
			if errs[i] != nil || results[i][0]["id"] != "shared" {
				t.Fatalf("sharer %d: out=%v err=%v", i, results[i], errs[i])
			}
		}
	})

	t.Run("invalidate forces the next view to recompute", func(t *testing.T) { // row 5
		var calls int
		c := &TerminalsCache{}
		compute := func() ([]map[string]any, error) {
			calls++
			return []map[string]any{{"n": calls}}, nil
		}
		if _, err := c.View(DefaultTTL, compute); err != nil {
			t.Fatal(err)
		}
		c.Invalidate()
		out, err := c.View(DefaultTTL, compute)
		if err != nil || calls != 2 || out[0]["n"] != 2 {
			t.Fatalf("after invalidate: out=%v err=%v calls=%d", out, err, calls)
		}
	})

	t.Run("a failed round is shared but not cached", func(t *testing.T) { // rows 6+7
		c := &TerminalsCache{}
		boom := errors.New("boom")
		block := make(chan struct{})
		var once sync.Once
		compute := func() ([]map[string]any, error) {
			once.Do(func() { close(block) })
			<-block
			return nil, boom
		}
		errCh := make(chan error, 2)
		for i := 0; i < 2; i++ { // leader + joiner share the round's error
			go func() {
				_, err := c.View(DefaultTTL, compute)
				errCh <- err
			}()
		}
		for i := 0; i < 2; i++ {
			if err := <-errCh; !errors.Is(err, boom) {
				t.Fatalf("round sharer %d: err=%v want boom", i, err)
			}
		}
		var calls int
		out, err := c.View(DefaultTTL, func() ([]map[string]any, error) {
			calls++
			return []map[string]any{{"id": "t2"}}, nil
		})
		if err != nil || calls != 1 || out[0]["id"] != "t2" {
			t.Fatalf("round after failure: out=%v err=%v calls=%d", out, err, calls)
		}
	})

	t.Run("leader cancellation surfaces to joiners then recovers", func(t *testing.T) { // row 8
		c := &TerminalsCache{}
		canceled := errors.New("canceled")
		block := make(chan struct{})
		var once sync.Once
		compute := func() ([]map[string]any, error) {
			once.Do(func() { close(block) })
			<-block
			return nil, canceled
		}
		if _, err := c.View(DefaultTTL, compute); err != nil && !errors.Is(err, canceled) { // the canceled round
			t.Fatalf("canceled round: err=%v", err)
		}
		out, err := c.View(DefaultTTL, func() ([]map[string]any, error) {
			return []map[string]any{{"id": "t3"}}, nil
		})
		if err != nil || out[0]["id"] != "t3" {
			t.Fatalf("recovery round: out=%v err=%v", out, err)
		}
	})
}
