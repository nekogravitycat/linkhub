package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// runPlan describes how a single scenario should be driven.
type runPlan struct {
	maxRequests int64         // >0 = run exactly this many attempts; 0 = use duration
	duration    time.Duration // used when maxRequests == 0
	warmup      time.Duration
	note        string
}

// limiter is a minimal open-loop rate limiter (stdlib only). It hands out
// evenly spaced "go" times so aggregate throughput targets cfg.Rate.
type limiter struct {
	mu       sync.Mutex
	interval time.Duration
	next     time.Time
}

func newLimiter(rate float64) *limiter {
	return &limiter{interval: time.Duration(float64(time.Second) / rate)}
}

// wait blocks until this caller's slot is due. Returns false if ctx is done.
func (l *limiter) wait(ctx context.Context) bool {
	l.mu.Lock()
	now := time.Now()
	if l.next.Before(now) {
		l.next = now
	}
	slot := l.next
	l.next = l.next.Add(l.interval)
	l.mu.Unlock()

	d := time.Until(slot)
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// runScenario drives `build` with cfg.Concurrency workers according to plan,
// excluding warmup-window requests from the reported metrics.
func runScenario(ctx context.Context, e *env, name string, build buildFunc, plan runPlan) Result {
	stats := make([]*workerStats, e.cfg.Concurrency)
	for i := range stats {
		stats[i] = newWorkerStats()
	}

	var lim *limiter
	if e.cfg.Rate > 0 {
		lim = newLimiter(e.cfg.Rate)
	}

	var counter atomic.Int64
	start := time.Now()
	warmupDeadline := start.Add(plan.warmup)
	var deadline time.Time
	if plan.maxRequests == 0 {
		deadline = start.Add(plan.duration)
	}

	var wg sync.WaitGroup
	for i := 0; i < e.cfg.Concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ws := stats[id]
			wc := &workerCtx{
				id:  id,
				rnd: rand.New(rand.NewSource(int64(id)*2862933555777941757 ^ start.UnixNano())),
			}
			for {
				if ctx.Err() != nil {
					return
				}
				if plan.maxRequests > 0 {
					if counter.Add(1) > plan.maxRequests {
						return
					}
				} else if !time.Now().Before(deadline) {
					return
				}
				if lim != nil && !lim.wait(ctx) {
					return
				}

				req, expect, err := build(e, wc)
				if err != nil {
					if errors.Is(err, errExhausted) {
						return
					}
					ws.requests++
					ws.addError(err.Error())
					continue
				}

				t0 := time.Now()
				resp, err := e.client.Do(req.WithContext(ctx))
				lat := time.Since(t0)

				// Requests issued during the warmup window are discarded.
				if t0.Before(warmupDeadline) {
					if err == nil {
						_, _ = io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
					}
					continue
				}

				ws.requests++
				ws.hist.record(float64(lat.Nanoseconds()))
				if err != nil {
					ws.addError(err.Error())
					continue
				}
				n, _ := io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				ws.bytes += n
				ws.statusCodes[resp.StatusCode]++
				if resp.StatusCode == expect {
					ws.success++
				} else {
					ws.addError(fmt.Sprintf("unexpected status %d (want %d)", resp.StatusCode, expect))
				}
			}
		}(i)
	}
	wg.Wait()

	window := time.Since(warmupDeadline)
	if window < 0 {
		window = 0
	}
	res := aggregate(name, e.cfg.Concurrency, window, stats)
	res.Note = plan.note
	return res
}
