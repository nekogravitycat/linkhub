package links

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

// cachedRepository decorates a Repository with a bounded, TTL'd in-memory cache
// for single-slug lookups (GetBySlug) — the hot redirect/get path. Only positive
// results are cached; writes invalidate the affected slug, and the TTL bounds how
// long a cached entry can be stale.
//
// Links are cached by value and copied on read, so callers (e.g. service.Update,
// which mutates the link it reads back) never share or corrupt the cached entry.
type cachedRepository struct {
	Repository
	cache *expirable.LRU[string, Link]
}

// NewCachedRepository wraps repo with a slug->Link cache. size bounds the number
// of entries and ttl bounds staleness; if either is non-positive the cache is
// disabled and repo is returned unchanged.
func NewCachedRepository(repo Repository, size int, ttl time.Duration) Repository {
	if size <= 0 || ttl <= 0 {
		return repo
	}
	return &cachedRepository{
		Repository: repo,
		cache:      expirable.NewLRU[string, Link](size, nil, ttl),
	}
}

func (r *cachedRepository) GetBySlug(ctx context.Context, slug string) (*Link, error) {
	if cached, ok := r.cache.Get(slug); ok {
		link := cached // copy so the caller cannot mutate the cached entry
		return &link, nil
	}

	link, err := r.Repository.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	r.cache.Add(slug, *link)
	return link, nil
}

func (r *cachedRepository) Update(ctx context.Context, link *Link) error {
	err := r.Repository.Update(ctx, link)
	// Invalidate on success, and also when the row is gone, so a stale entry is
	// never served after the underlying state changed.
	if err == nil || errors.Is(err, ErrLinkNotFound) {
		r.cache.Remove(link.Slug)
	}
	return err
}

func (r *cachedRepository) Delete(ctx context.Context, slug string) error {
	err := r.Repository.Delete(ctx, slug)
	if err == nil || errors.Is(err, ErrLinkNotFound) {
		r.cache.Remove(slug)
	}
	return err
}
