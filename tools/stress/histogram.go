package main

import "math"

// Log-bucketed latency histogram. Bucket i covers [base^i, base^(i+1)) ns, so
// memory is O(buckets) regardless of how many samples are recorded — this lets
// duration-based runs record an unbounded number of requests with bounded
// memory. Resolution is ~5% (base 1.05), which is plenty for p50..p99.
const (
	histBase    = 1.05
	histBuckets = 512 // covers ~1ns .. ~72s
)

var histLogBase = math.Log(histBase)

type histogram struct {
	counts [histBuckets]int64
	count  int64
	sum    float64 // sum of all recorded ns, for an exact mean
	min    float64
	max    float64
}

func bucketIndex(ns float64) int {
	if ns <= 1 {
		return 0
	}
	i := int(math.Log(ns) / histLogBase)
	if i < 0 {
		return 0
	}
	if i >= histBuckets {
		return histBuckets - 1
	}
	return i
}

func bucketUpperBound(i int) float64 {
	return math.Pow(histBase, float64(i+1))
}

func (h *histogram) record(ns float64) {
	if h.count == 0 || ns < h.min {
		h.min = ns
	}
	if ns > h.max {
		h.max = ns
	}
	h.counts[bucketIndex(ns)]++
	h.count++
	h.sum += ns
}

func (h *histogram) merge(o *histogram) {
	if o.count == 0 {
		return
	}
	if h.count == 0 || o.min < h.min {
		h.min = o.min
	}
	if o.max > h.max {
		h.max = o.max
	}
	for i := range o.counts {
		h.counts[i] += o.counts[i]
	}
	h.count += o.count
	h.sum += o.sum
}

func (h *histogram) mean() float64 {
	if h.count == 0 {
		return 0
	}
	return h.sum / float64(h.count)
}

// percentile returns the latency (ns) at the given percentile (0..100). The
// value is the upper bound of the bucket the percentile falls into, clamped to
// the observed max so we never report a value larger than anything seen.
func (h *histogram) percentile(p float64) float64 {
	if h.count == 0 {
		return 0
	}
	target := int64(math.Ceil(p / 100 * float64(h.count)))
	if target < 1 {
		target = 1
	}
	var cum int64
	for i := 0; i < histBuckets; i++ {
		cum += h.counts[i]
		if cum >= target {
			if ub := bucketUpperBound(i); ub < h.max {
				return ub
			}
			return h.max
		}
	}
	return h.max
}
