// Command stress is a load-testing tool for the linkhub backend. It drives the
// API's operations (create, get, list, update, redirect, delete, and a mixed
// workload) at high concurrency and reports throughput and latency percentiles.
//
// See README.md for usage and examples.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
		os.Exit(1)
	}
}

func run(cfg Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	prefix := "s" + strconv.FormatInt(time.Now().UnixNano(), 36)
	e := &env{
		cfg:    cfg,
		client: newHTTPClient(cfg),
		slugs:  newSlugGen(prefix),
	}

	scenarios := cfg.scenarios()
	printHeader(cfg, prefix, scenarios)

	// With --db-dsn, the table is reset and reseeded with a standard pool before
	// every scenario (setup/teardown over the DB; load stays over HTTP), so each
	// scenario is measured from an identical, known state — results are
	// independent of scenario order and comparable across runs. Without it, the
	// pools are seeded once over HTTP (legacy behavior).
	var fx *fixture
	if cfg.DBDSN != "" {
		var err error
		fx, err = newFixture(ctx, cfg.DBDSN)
		if err != nil {
			return fmt.Errorf("open --db-dsn fixture (is Postgres reachable?): %w", err)
		}
		defer fx.close()
		fmt.Println("  fixture      db-assisted: reset + standard reseed before each scenario")
		fmt.Println()
	} else if err := seedHTTPPools(ctx, e, scenarios); err != nil {
		return err
	}

	var results []Result
	for _, name := range scenarios {
		if ctx.Err() != nil {
			fmt.Println("\ninterrupted — stopping early")
			break
		}
		if fx != nil {
			if err := fx.prepare(ctx, e, name); err != nil {
				return fmt.Errorf("fixture prepare for %q: %w", name, err)
			}
		}
		plan := planFor(name, cfg, len(e.delPool))
		res := runScenario(ctx, e, name, scenarioBuild(name), plan)
		printResult(res)
		results = append(results, res)
	}

	printComparison(results)

	if cfg.Out != "" {
		if err := exportJSON(cfg.Out, results); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write JSON: %v\n", err)
		} else {
			fmt.Printf("\nwrote %s\n", cfg.Out)
		}
	}
	if cfg.CSV != "" {
		if err := exportCSV(cfg.CSV, results); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write CSV: %v\n", err)
		} else {
			fmt.Printf("wrote %s\n", cfg.CSV)
		}
	}

	if fx != nil {
		if err := fx.reset(context.WithoutCancel(ctx)); err != nil {
			fmt.Fprintf(os.Stderr, "\nfixture: final reset failed: %v\n", err)
		} else {
			fmt.Println("\nfixture: table reset to empty")
		}
	} else {
		cleanup(ctx, e, prefix)
	}
	return nil
}

// seedHTTPPools seeds the read and/or delete pools once over HTTP (legacy path
// used when --db-dsn is not provided).
func seedHTTPPools(ctx context.Context, e *env, scenarios []string) error {
	readNeeded := false
	delNeeded := false
	for _, name := range scenarios {
		if needsReadPool(name) {
			readNeeded = true
		}
		if name == "delete" {
			delNeeded = true
		}
	}

	if readNeeded {
		if e.cfg.Seed < 1 {
			return fmt.Errorf("scenarios %v need seeded links; set --seed >= 1", scenarios)
		}
		slugs, err := seedLinks(ctx, e, e.cfg.Seed, "read pool")
		if err != nil {
			return fmt.Errorf("seeding read pool failed (is the backend up?): %w", err)
		}
		e.readPool = slugs
	}

	if delNeeded {
		delN := e.cfg.Seed
		if e.cfg.Requests > 0 {
			delN = int(e.cfg.Requests)
		}
		if delN < 1 {
			return fmt.Errorf("the delete scenario needs --seed or --requests >= 1")
		}
		slugs, err := seedLinks(ctx, e, delN, "delete pool")
		if err != nil {
			return fmt.Errorf("seeding delete pool failed (is the backend up?): %w", err)
		}
		e.delPool = slugs
	}
	return nil
}

// planFor builds the run plan for a scenario. delete is bounded by its pool
// size; in --requests mode warmup is skipped so the measured count is exact.
func planFor(name string, cfg Config, delPoolLen int) runPlan {
	switch {
	case name == "delete":
		return runPlan{
			maxRequests: int64(delPoolLen),
			note:        fmt.Sprintf("bounded to %d pre-seeded links (ignores --duration)", delPoolLen),
		}
	case cfg.Requests > 0:
		p := runPlan{maxRequests: cfg.Requests}
		p.note = mixedNote(name)
		return p
	default:
		p := runPlan{duration: cfg.Duration, warmup: cfg.Warmup}
		p.note = mixedNote(name)
		return p
	}
}

func mixedNote(name string) string {
	if name == "mixed" {
		return "weighted mix: 75% redirect, 15% get, 5% list, 3% create, 2% update"
	}
	return ""
}

// cleanup removes the seeded read pool. Links created by the create/mixed
// scenarios are intentionally left in place (their count is unbounded); the run
// prefix is printed so they can be purged in bulk.
func cleanup(ctx context.Context, e *env, prefix string) {
	if e.cfg.NoCleanup {
		fmt.Printf("\nskipping cleanup (--no-cleanup). Seeded/created links use prefix %q.\n", prefix)
		return
	}
	if len(e.readPool) > 0 && ctx.Err() == nil {
		fmt.Printf("\ncleaning up %d seeded links...\n", len(e.readPool))
		deleted := cleanupLinks(context.WithoutCancel(ctx), e, e.readPool)
		fmt.Printf("  deleted %d\n", deleted)
	}
	fmt.Printf("note: links created by create/mixed are not auto-deleted. "+
		"To purge: DELETE FROM links WHERE slug LIKE '%s%%'; (or TRUNCATE links).\n", prefix)
}

func printHeader(cfg Config, prefix string, scenarios []string) {
	mode := fmt.Sprintf("duration %s", cfg.Duration)
	if cfg.Requests > 0 {
		mode = fmt.Sprintf("requests %d", cfg.Requests)
	}
	rate := "unlimited (closed-loop)"
	if cfg.Rate > 0 {
		rate = fmt.Sprintf("%.0f req/s (open-loop)", cfg.Rate)
	}
	fmt.Println("linkhub stress test")
	fmt.Printf("  target       %s\n", cfg.URL)
	fmt.Printf("  scenarios    %v\n", scenarios)
	fmt.Printf("  concurrency  %d\n", cfg.Concurrency)
	fmt.Printf("  mode         %s\n", mode)
	fmt.Printf("  rate         %s\n", rate)
	fmt.Printf("  warmup       %s\n", cfg.Warmup)
	fmt.Printf("  run prefix   %s\n", prefix)
	fmt.Println()
}
