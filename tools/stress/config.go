package main

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

// Config holds all tunables, populated from command-line flags.
type Config struct {
	URL            string
	Scenario       string
	Concurrency    int
	Duration       time.Duration
	Requests       int64
	Rate           float64
	Seed           int
	Warmup         time.Duration
	Timeout        time.Duration
	Conns          int
	RedirectDomain string
	Out            string
	CSV            string
	KeepAlive      bool
	NoCleanup      bool
}

// validScenarios are the names accepted by --scenario (besides "all").
var validScenarios = []string{"create", "get", "list", "update", "redirect", "delete", "mixed"}

func parseConfig(args []string) (Config, error) {
	var cfg Config
	fs := flag.NewFlagSet("stress", flag.ContinueOnError)

	fs.StringVar(&cfg.URL, "url", "http://localhost:8001", "base URL of the backend API (point at nginx ports to include its rate limits)")
	fs.StringVar(&cfg.Scenario, "scenario", "all", "scenario to run: all|"+strings.Join(validScenarios, "|"))

	fs.IntVar(&cfg.Concurrency, "concurrency", 50, "number of concurrent workers")
	fs.IntVar(&cfg.Concurrency, "c", 50, "shorthand for --concurrency")

	fs.DurationVar(&cfg.Duration, "duration", 30*time.Second, "how long to run each scenario (ignored when --requests is set)")
	fs.DurationVar(&cfg.Duration, "d", 30*time.Second, "shorthand for --duration")

	fs.Int64Var(&cfg.Requests, "requests", 0, "total requests per scenario (0 = use --duration)")
	fs.Int64Var(&cfg.Requests, "n", 0, "shorthand for --requests")

	fs.Float64Var(&cfg.Rate, "rate", 0, "target requests/sec (0 = closed-loop, as fast as possible)")
	fs.IntVar(&cfg.Seed, "seed", 10000, "links to pre-create for read/update/redirect/delete scenarios")
	fs.DurationVar(&cfg.Warmup, "warmup", 2*time.Second, "warmup window whose requests are excluded from metrics")
	fs.DurationVar(&cfg.Timeout, "timeout", 10*time.Second, "per-request timeout")
	fs.IntVar(&cfg.Conns, "conns", 0, "max idle connections per host (0 = 2x concurrency)")
	fs.StringVar(&cfg.RedirectDomain, "redirect-domain", "localhost:8003", "domain excluded from generated target URLs (must match backend REDIRECT_DOMAIN)")
	fs.StringVar(&cfg.Out, "out", "", "write per-scenario results to this JSON file")
	fs.StringVar(&cfg.CSV, "csv", "", "write per-scenario results to this CSV file")
	fs.BoolVar(&cfg.KeepAlive, "keepalive", true, "use HTTP keep-alive (set false to measure cold-connection cost)")
	fs.BoolVar(&cfg.NoCleanup, "no-cleanup", false, "skip deleting seeded links at the end")

	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	if c.URL == "" {
		return fmt.Errorf("--url must not be empty")
	}
	c.URL = strings.TrimRight(c.URL, "/")
	if c.Scenario != "all" && !contains(validScenarios, c.Scenario) {
		return fmt.Errorf("invalid --scenario %q (want all|%s)", c.Scenario, strings.Join(validScenarios, "|"))
	}
	if c.Concurrency < 1 {
		return fmt.Errorf("--concurrency must be >= 1")
	}
	if c.Requests < 0 {
		return fmt.Errorf("--requests must be >= 0")
	}
	if c.Requests == 0 && c.Duration <= 0 {
		return fmt.Errorf("either --duration or --requests must be positive")
	}
	if c.Rate < 0 {
		return fmt.Errorf("--rate must be >= 0")
	}
	if c.Seed < 0 {
		return fmt.Errorf("--seed must be >= 0")
	}
	if c.Conns < 0 {
		return fmt.Errorf("--conns must be >= 0")
	}
	return nil
}

// scenarios returns the ordered list of scenarios to run. Order matters in
// "all" mode: redirect/get run while the seeded pool is pristine, update keeps
// links active, and delete runs last because it consumes its own pool.
func (c *Config) scenarios() []string {
	if c.Scenario != "all" {
		return []string{c.Scenario}
	}
	return []string{"create", "get", "list", "redirect", "update", "mixed", "delete"}
}

// maxIdleConns resolves the effective connection-pool size.
func (c *Config) maxIdleConns() int {
	if c.Conns > 0 {
		return c.Conns
	}
	return 2 * c.Concurrency
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
