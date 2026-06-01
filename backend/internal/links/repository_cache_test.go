package links

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRepo is an in-memory Repository for testing the cache decorator without a
// database. It counts GetBySlug calls so tests can assert cache hits/misses.
type fakeRepo struct {
	links    map[string]*Link
	getCalls int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{links: map[string]*Link{}}
}

func (f *fakeRepo) Create(ctx context.Context, slug, url string) error {
	if _, ok := f.links[slug]; ok {
		return ErrSlugTaken
	}
	f.links[slug] = &Link{Slug: slug, URL: url, IsActive: true}
	return nil
}

func (f *fakeRepo) GetBySlug(ctx context.Context, slug string) (*Link, error) {
	f.getCalls++
	l, ok := f.links[slug]
	if !ok {
		return nil, ErrLinkNotFound
	}
	cp := *l // return a copy, like the real repo
	return &cp, nil
}

func (f *fakeRepo) Update(ctx context.Context, link *Link) error {
	if _, ok := f.links[link.Slug]; !ok {
		return ErrLinkNotFound
	}
	cp := *link
	f.links[link.Slug] = &cp
	return nil
}

func (f *fakeRepo) Delete(ctx context.Context, slug string) error {
	if _, ok := f.links[slug]; !ok {
		return ErrLinkNotFound
	}
	delete(f.links, slug)
	return nil
}

func (f *fakeRepo) List(ctx context.Context, opts ListOptions) ([]*Link, int64, error) {
	return nil, 0, nil
}

func TestCachedRepository_GetCachesPositiveLookups(t *testing.T) {
	ctx := context.Background()
	fake := newFakeRepo()
	require.NoError(t, fake.Create(ctx, "abc", "https://example.com"))

	repo := NewCachedRepository(fake, 100, time.Minute)

	first, err := repo.GetBySlug(ctx, "abc")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", first.URL)

	second, err := repo.GetBySlug(ctx, "abc")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", second.URL)

	assert.Equal(t, 1, fake.getCalls, "second lookup should be served from cache")
}

func TestCachedRepository_DoesNotCacheMisses(t *testing.T) {
	ctx := context.Background()
	fake := newFakeRepo()
	repo := NewCachedRepository(fake, 100, time.Minute)

	_, err := repo.GetBySlug(ctx, "missing")
	require.ErrorIs(t, err, ErrLinkNotFound)
	_, err = repo.GetBySlug(ctx, "missing")
	require.ErrorIs(t, err, ErrLinkNotFound)

	assert.Equal(t, 2, fake.getCalls, "misses must not be cached")
}

func TestCachedRepository_UpdateInvalidates(t *testing.T) {
	ctx := context.Background()
	fake := newFakeRepo()
	require.NoError(t, fake.Create(ctx, "abc", "https://old.example"))
	repo := NewCachedRepository(fake, 100, time.Minute)

	_, err := repo.GetBySlug(ctx, "abc") // fill cache
	require.NoError(t, err)

	require.NoError(t, repo.Update(ctx, &Link{Slug: "abc", URL: "https://new.example", IsActive: false}))

	got, err := repo.GetBySlug(ctx, "abc")
	require.NoError(t, err)
	assert.Equal(t, "https://new.example", got.URL, "update must invalidate the cache")
	assert.False(t, got.IsActive)
	assert.Equal(t, 2, fake.getCalls, "post-update read should miss the cache")
}

func TestCachedRepository_DeleteInvalidates(t *testing.T) {
	ctx := context.Background()
	fake := newFakeRepo()
	require.NoError(t, fake.Create(ctx, "abc", "https://example.com"))
	repo := NewCachedRepository(fake, 100, time.Minute)

	_, err := repo.GetBySlug(ctx, "abc") // fill cache
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, "abc"))

	_, err = repo.GetBySlug(ctx, "abc")
	assert.ErrorIs(t, err, ErrLinkNotFound, "delete must invalidate the cache")
}

func TestCachedRepository_TTLExpiry(t *testing.T) {
	ctx := context.Background()
	fake := newFakeRepo()
	require.NoError(t, fake.Create(ctx, "abc", "https://example.com"))
	repo := NewCachedRepository(fake, 100, 50*time.Millisecond)

	_, err := repo.GetBySlug(ctx, "abc")
	require.NoError(t, err)

	time.Sleep(120 * time.Millisecond)

	_, err = repo.GetBySlug(ctx, "abc")
	require.NoError(t, err)
	assert.Equal(t, 2, fake.getCalls, "entry should expire after the TTL and re-fetch")
}

func TestCachedRepository_ReturnedLinkIsNotShared(t *testing.T) {
	ctx := context.Background()
	fake := newFakeRepo()
	require.NoError(t, fake.Create(ctx, "abc", "https://example.com"))
	repo := NewCachedRepository(fake, 100, time.Minute)

	first, err := repo.GetBySlug(ctx, "abc")
	require.NoError(t, err)
	first.URL = "https://mutated.example" // caller mutates its copy

	second, err := repo.GetBySlug(ctx, "abc")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", second.URL, "mutating a returned link must not corrupt the cache")
}

func TestNewCachedRepository_DisabledWhenSizeOrTTLZero(t *testing.T) {
	fake := newFakeRepo()
	assert.Same(t, Repository(fake), NewCachedRepository(fake, 0, time.Minute), "size<=0 disables caching")
	assert.Same(t, Repository(fake), NewCachedRepository(fake, 100, 0), "ttl<=0 disables caching")
}
