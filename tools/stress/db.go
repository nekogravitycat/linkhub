package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// fixture manages the database state for a run via a direct Postgres connection.
// It is used ONLY for setup/teardown — resetting the table and bulk-seeding a
// standard pool between scenarios. All measured load still goes over HTTP; the
// DB connection never serves traffic that is timed.
//
// This lets every scenario start from an identical, known table state, so a
// scenario's result no longer depends on how much data a previous scenario
// (e.g. create) inserted, and runs stay comparable across invocations.
type fixture struct {
	pool *pgxpool.Pool
}

func newFixture(ctx context.Context, dsn string) (*fixture, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &fixture{pool: pool}, nil
}

func (f *fixture) close() {
	f.pool.Close()
}

// reset returns the links table to a known-empty state.
func (f *fixture) reset(ctx context.Context) error {
	if _, err := f.pool.Exec(ctx, "TRUNCATE links RESTART IDENTITY"); err != nil {
		return fmt.Errorf("truncate links: %w", err)
	}
	return nil
}

// seed bulk-inserts n standard, active links via COPY and returns their slugs.
//
// Slugs are "<prefix>_<i>" (underscore separator) so they never collide with the
// "<prefix>-<base36>" slugs the create/mixed scenarios generate over HTTP. They
// stay within the backend's slug rules (^[a-zA-Z0-9-_]+$, <=32 chars) for the
// run prefix plus realistic seed sizes, so read scenarios can look them up.
func (f *fixture) seed(ctx context.Context, prefix string, n int) ([]string, error) {
	if n <= 0 {
		return nil, nil
	}
	slugs := make([]string, n)
	rows := make([][]any, n)
	for i := range slugs {
		slug := prefix + "_" + strconv.Itoa(i)
		slugs[i] = slug
		rows[i] = []any{slug, "https://example.com/" + slug, true}
	}
	_, err := f.pool.CopyFrom(ctx,
		pgx.Identifier{"links"},
		[]string{"slug", "url", "is_active"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return nil, fmt.Errorf("copy %d seed rows: %w", n, err)
	}
	return slugs, nil
}

// prepare resets the table and reseeds a standard pool sized for the scenario,
// wiring the pool into the env. delete consumes its pool, so its index is reset.
func (f *fixture) prepare(ctx context.Context, e *env, name string) error {
	if err := f.reset(ctx); err != nil {
		return err
	}

	n := e.cfg.Seed
	if name == "delete" && e.cfg.Requests > 0 {
		n = int(e.cfg.Requests)
	}
	if (needsReadPool(name) || name == "delete") && n < 1 {
		return fmt.Errorf("scenario %q needs a seeded pool; set --seed >= 1", name)
	}

	slugs, err := f.seed(ctx, e.slugs.prefix, n)
	if err != nil {
		return err
	}
	e.readPool = slugs
	e.delPool = slugs
	e.delIdx.Store(0)

	fmt.Printf("  %-8s standard pool: %d links\n", name, len(slugs))
	return nil
}
