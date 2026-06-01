package main

import "time"

const maxErrSamplesPerWorker = 5

// workerStats is owned by a single worker goroutine, so it needs no locking on
// the hot path. Stats are merged once at the end of a scenario.
type workerStats struct {
	hist        histogram
	statusCodes map[int]int64
	requests    int64
	success     int64
	errors      int64
	bytes       int64
	errSamples  []string
}

func newWorkerStats() *workerStats {
	return &workerStats{statusCodes: make(map[int]int64)}
}

func (w *workerStats) addError(msg string) {
	w.errors++
	if len(w.errSamples) < maxErrSamplesPerWorker {
		w.errSamples = append(w.errSamples, msg)
	}
}

// Result is the aggregated outcome of one scenario. Latencies are nanoseconds.
type Result struct {
	Scenario    string           `json:"scenario"`
	Concurrency int              `json:"concurrency"`
	WindowSec   float64          `json:"window_sec"`
	Requests    int64            `json:"requests"`
	Success     int64            `json:"success"`
	Errors      int64            `json:"errors"`
	Bytes       int64            `json:"bytes"`
	RPS         float64          `json:"rps"`
	MinMs       float64          `json:"min_ms"`
	MeanMs      float64          `json:"mean_ms"`
	P50Ms       float64          `json:"p50_ms"`
	P90Ms       float64          `json:"p90_ms"`
	P95Ms       float64          `json:"p95_ms"`
	P99Ms       float64          `json:"p99_ms"`
	P999Ms      float64          `json:"p999_ms"`
	MaxMs       float64          `json:"max_ms"`
	StatusCodes map[int]int64    `json:"status_codes"`
	ErrSamples  []string         `json:"error_samples,omitempty"`
	Note        string           `json:"note,omitempty"`
}

func aggregate(scenario string, concurrency int, window time.Duration, stats []*workerStats) Result {
	var h histogram
	codes := make(map[int]int64)
	var req, succ, errs, bytes int64
	var samples []string
	for _, ws := range stats {
		h.merge(&ws.hist)
		for code, n := range ws.statusCodes {
			codes[code] += n
		}
		req += ws.requests
		succ += ws.success
		errs += ws.errors
		bytes += ws.bytes
		for _, s := range ws.errSamples {
			if len(samples) < 20 {
				samples = append(samples, s)
			}
		}
	}

	rps := 0.0
	if window > 0 {
		rps = float64(succ) / window.Seconds()
	}

	return Result{
		Scenario:    scenario,
		Concurrency: concurrency,
		WindowSec:   window.Seconds(),
		Requests:    req,
		Success:     succ,
		Errors:      errs,
		Bytes:       bytes,
		RPS:         rps,
		MinMs:       nsToMs(h.min),
		MeanMs:      nsToMs(h.mean()),
		P50Ms:       nsToMs(h.percentile(50)),
		P90Ms:       nsToMs(h.percentile(90)),
		P95Ms:       nsToMs(h.percentile(95)),
		P99Ms:       nsToMs(h.percentile(99)),
		P999Ms:      nsToMs(h.percentile(99.9)),
		MaxMs:       nsToMs(h.max),
		StatusCodes: codes,
		ErrSamples:  samples,
	}
}

func nsToMs(ns float64) float64 { return ns / 1e6 }
