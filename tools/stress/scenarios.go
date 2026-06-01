package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// errExhausted is returned by a build func when its single-use pool is empty.
// The runner treats it as a clean stop signal, not an error.
var errExhausted = errors.New("pool exhausted")

// env holds resources shared by all workers for the duration of a run.
type env struct {
	cfg      Config
	client   *http.Client
	slugs    *slugGen
	readPool []string // persistent pool for get/list/update/redirect/mixed
	delPool  []string // single-use pool consumed by the delete scenario
	delIdx   atomic.Int64
}

// workerCtx is per-worker state (a non-shared RNG) passed to build funcs.
type workerCtx struct {
	id  int
	rnd *rand.Rand
}

// buildFunc constructs the next request for a worker and returns the HTTP
// status that counts as success.
type buildFunc func(e *env, w *workerCtx) (*http.Request, int, error)

func (e *env) pickRead(w *workerCtx) string {
	return e.readPool[w.rnd.Intn(len(e.readPool))]
}

func jsonRequest(method, url string, payload any) (*http.Request, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// targetURL builds a valid external target URL that never contains the
// backend's redirect domain (which would trigger a 400 loop-detection error).
func (e *env) targetURL(suffix string) string {
	return "https://example.com/" + suffix
}

func buildCreate(e *env, w *workerCtx) (*http.Request, int, error) {
	slug := e.slugs.next()
	req, err := jsonRequest(http.MethodPost, e.cfg.URL+"/links",
		map[string]string{"slug": slug, "url": e.targetURL(slug)})
	return req, http.StatusCreated, err
}

func buildGet(e *env, w *workerCtx) (*http.Request, int, error) {
	req, err := http.NewRequest(http.MethodGet, e.cfg.URL+"/links/"+e.pickRead(w), nil)
	return req, http.StatusOK, err
}

func buildList(e *env, w *workerCtx) (*http.Request, int, error) {
	q := url.Values{}
	q.Set("page", strconv.Itoa(1+w.rnd.Intn(5)))
	q.Set("page_size", []string{"10", "20", "50", "100"}[w.rnd.Intn(4)])
	q.Set("sort_by", []string{"created_at", "updated_at", "slug", "id"}[w.rnd.Intn(4)])
	q.Set("sort_order", []string{"asc", "desc"}[w.rnd.Intn(2)])
	// Half the time, search by the run prefix so the keyword path actually
	// matches seeded rows (prefix is >3 chars, satisfying the validator).
	if w.rnd.Intn(2) == 0 {
		q.Set("keyword", e.slugs.prefix)
	}
	if w.rnd.Intn(3) == 0 {
		q.Set("is_active", "true")
	}
	req, err := http.NewRequest(http.MethodGet, e.cfg.URL+"/links?"+q.Encode(), nil)
	return req, http.StatusOK, err
}

func buildUpdate(e *env, w *workerCtx) (*http.Request, int, error) {
	// Keep links active so they stay valid for the redirect scenario in
	// "all" mode; still exercises the full PATCH/DB-update/trigger path.
	payload := map[string]any{
		"url":       e.targetURL("u/" + strconv.Itoa(w.rnd.Int())),
		"is_active": true,
	}
	req, err := jsonRequest(http.MethodPatch, e.cfg.URL+"/links/"+e.pickRead(w), payload)
	return req, http.StatusOK, err
}

func buildRedirect(e *env, w *workerCtx) (*http.Request, int, error) {
	req, err := http.NewRequest(http.MethodGet, e.cfg.URL+"/redirect/"+e.pickRead(w), nil)
	return req, http.StatusFound, err // 302
}

func buildDelete(e *env, w *workerCtx) (*http.Request, int, error) {
	i := e.delIdx.Add(1) - 1
	if i >= int64(len(e.delPool)) {
		return nil, 0, errExhausted
	}
	req, err := http.NewRequest(http.MethodDelete, e.cfg.URL+"/links/"+e.delPool[i], nil)
	return req, http.StatusOK, err
}

// buildMixed approximates real traffic: read-heavy with a few writes.
func buildMixed(e *env, w *workerCtx) (*http.Request, int, error) {
	switch r := w.rnd.Intn(100); {
	case r < 75:
		return buildRedirect(e, w)
	case r < 90:
		return buildGet(e, w)
	case r < 95:
		return buildList(e, w)
	case r < 98:
		return buildCreate(e, w)
	default:
		return buildUpdate(e, w)
	}
}

func scenarioBuild(name string) buildFunc {
	switch name {
	case "create":
		return buildCreate
	case "get":
		return buildGet
	case "list":
		return buildList
	case "update":
		return buildUpdate
	case "redirect":
		return buildRedirect
	case "delete":
		return buildDelete
	case "mixed":
		return buildMixed
	default:
		return nil
	}
}

// needsReadPool reports whether a scenario reads from the persistent pool.
func needsReadPool(name string) bool {
	switch name {
	case "get", "list", "update", "redirect", "mixed":
		return true
	}
	return false
}

// seedLinks creates `count` links concurrently and returns their slugs. It
// aborts on the first persistent failure so the operator immediately sees a
// dead backend / DB rather than a flood of errors.
func seedLinks(ctx context.Context, e *env, count int, label string) ([]string, error) {
	if count <= 0 {
		return nil, nil
	}
	slugs := make([]string, count)
	var idx, done atomic.Int64
	var failed atomic.Bool
	var mu sync.Mutex
	var firstErr error
	setErr := func(err error) {
		mu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		mu.Unlock()
		failed.Store(true)
	}

	workers := e.cfg.Concurrency
	if workers > count {
		workers = count
	}

	progressDone := make(chan struct{})
	go func() {
		t := time.NewTicker(500 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-progressDone:
				return
			case <-t.C:
				fmt.Printf("\r  seeding %s: %d/%d", label, done.Load(), count)
			}
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if failed.Load() || ctx.Err() != nil {
					return
				}
				j := idx.Add(1) - 1
				if j >= int64(count) {
					return
				}
				slug := e.slugs.next()
				req, err := jsonRequest(http.MethodPost, e.cfg.URL+"/links",
					map[string]string{"slug": slug, "url": e.targetURL(slug)})
				if err != nil {
					setErr(err)
					return
				}
				resp, err := e.client.Do(req.WithContext(ctx))
				if err != nil {
					setErr(fmt.Errorf("seed POST /links: %w", err))
					return
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if resp.StatusCode != http.StatusCreated {
					setErr(fmt.Errorf("seed POST /links returned %d (want 201)", resp.StatusCode))
					return
				}
				slugs[j] = slug
				done.Add(1)
			}
		}()
	}
	wg.Wait()
	close(progressDone)
	fmt.Printf("\r  seeding %s: %d/%d\n", label, done.Load(), count)

	if firstErr != nil {
		return nil, firstErr
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return slugs, nil
}

// cleanupLinks best-effort deletes the given slugs. Errors are ignored — the
// backend may already have removed some (e.g. via the delete scenario).
func cleanupLinks(ctx context.Context, e *env, slugs []string) int64 {
	if len(slugs) == 0 {
		return 0
	}
	var idx, deleted atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < e.cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if ctx.Err() != nil {
					return
				}
				j := idx.Add(1) - 1
				if j >= int64(len(slugs)) {
					return
				}
				req, err := http.NewRequest(http.MethodDelete, e.cfg.URL+"/links/"+slugs[j], nil)
				if err != nil {
					continue
				}
				resp, err := e.client.Do(req.WithContext(ctx))
				if err != nil {
					continue
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					deleted.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	return deleted.Load()
}
