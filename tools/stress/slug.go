package main

import (
	"strconv"
	"sync/atomic"
)

// slugGen produces unique, regex-safe slugs (^[a-zA-Z0-9-_]+$, 1-32 chars).
//
// Every slug is "<prefix>-<n>" where n is a process-global atomic counter
// rendered in base36. A single generator is shared across all goroutines and
// pools, so no two slugs ever collide within a run. The prefix embeds a
// per-run component so repeated runs against the same database don't clash and
// so seeded data can be identified/purged by prefix.
type slugGen struct {
	prefix string
	n      atomic.Int64
}

func newSlugGen(prefix string) *slugGen {
	return &slugGen{prefix: prefix}
}

// next returns the next unique slug. base36 keeps it within the 32-char limit
// even for billions of requests (e.g. 1e12 -> ~8 chars).
func (g *slugGen) next() string {
	i := g.n.Add(1)
	return g.prefix + "-" + strconv.FormatInt(i, 36)
}
