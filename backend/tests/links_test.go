package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nekogravitycat/linkhub/internal/links"
	"github.com/nekogravitycat/linkhub/internal/pkg/request"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinksRepository(t *testing.T) {
	if testPool == nil {
		t.Skip("Postgres test pool not initialized (POSTGRES_TEST_DB not set)")
	}

	ctx := context.Background()
	repo := links.NewRepository(testPool)

	t.Run("Create and Get Link", func(t *testing.T) {
		slug := "test-google"
		url := "https://google.com"

		// Create
		err := repo.Create(ctx, slug, url)
		require.NoError(t, err)

		// Get
		link, err := repo.GetBySlug(ctx, slug)
		require.NoError(t, err)

		assert.Equal(t, slug, link.Slug)
		assert.Equal(t, url, link.URL)
		assert.True(t, link.IsActive)
	})

	t.Run("Update Link", func(t *testing.T) {
		slug := "test-update"
		url := "https://original.com"

		err := repo.Create(ctx, slug, url)
		require.NoError(t, err)

		link, err := repo.GetBySlug(ctx, slug)
		require.NoError(t, err)

		// Update
		newURL := "https://updated.com"
		link.URL = newURL
		link.IsActive = false

		err = repo.Update(ctx, link)
		require.NoError(t, err)

		// Verify
		updatedLink, err := repo.GetBySlug(ctx, slug)
		require.NoError(t, err)

		assert.Equal(t, newURL, updatedLink.URL)
		assert.False(t, updatedLink.IsActive)
	})

	t.Run("Delete Link", func(t *testing.T) {
		slug := "test-delete"
		url := "https://todelete.com"

		err := repo.Create(ctx, slug, url)
		require.NoError(t, err)

		// Delete
		err = repo.Delete(ctx, slug)
		require.NoError(t, err)

		// Verify
		_, err = repo.GetBySlug(ctx, slug)
		assert.ErrorIs(t, err, links.ErrLinkNotFound)
	})

	t.Run("List Links", func(t *testing.T) {
		slug1 := "list-1"
		slug2 := "list-2"

		// We use require.NoError for setup steps to fail fast if setup fails
		require.NoError(t, repo.Create(ctx, slug1, "http://1.com"))

		time.Sleep(time.Millisecond * 10)
		require.NoError(t, repo.Create(ctx, slug2, "http://2.com"))

		list, total, err := repo.List(ctx, links.ListOptions{})
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(list), 2)
		assert.Equal(t, int64(len(list)), total)

		// Check if slug2 exists in list
		foundSlug2 := false
		for _, l := range list {
			if l.Slug == slug2 {
				foundSlug2 = true
				break
			}
		}
		assert.True(t, foundSlug2, "Did not find slug %s in list", slug2)

	})

	t.Run("List Links with Search and Filter", func(t *testing.T) {
		// Clean slate for this test potentially, or just use distinct prefixes
		p := "search-" + time.Now().Format("150405")

		// 1. Active, matches keyword
		_ = repo.Create(ctx, p+"-apple", "http://apple.com")

		// 2. Inactive, matches keyword
		_ = repo.Create(ctx, p+"-banana", "http://banana.com")
		l, _ := repo.GetBySlug(ctx, p+"-banana")
		l.IsActive = false
		_ = repo.Update(ctx, l)

		// 3. Active, no match
		_ = repo.Create(ctx, p+"-carrot", "http://carrot.com")

		// Test Keyword Search (should find apple and banana)
		list, total, err := repo.List(ctx, links.ListOptions{
			Keyword: p, // Should match all with prefix
		})
		require.NoError(t, err)
		assert.Equal(t, 3, len(list))
		assert.Equal(t, int64(3), total)

		list, total, err = repo.List(ctx, links.ListOptions{
			Keyword: "apple",
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(list))
		assert.Equal(t, p+"-apple", list[0].Slug)

		// Test IsActive Filter (active only)
		active := true
		list, total, err = repo.List(ctx, links.ListOptions{
			Keyword:  p,
			IsActive: &active,
		})
		require.NoError(t, err)
		// Should find apple and carrot
		assert.Equal(t, 2, len(list))

		// Test IsActive Filter (inactive only)
		inactive := false
		list, total, err = repo.List(ctx, links.ListOptions{
			Keyword:  p,
			IsActive: &inactive,
		})
		require.NoError(t, err)
		// Should find banana only
		assert.Equal(t, 1, len(list))
		assert.Equal(t, p+"-banana", list[0].Slug)

		// Test Combo (inactive + keyword "banana")
		list, total, err = repo.List(ctx, links.ListOptions{
			Keyword:  "banana",
			IsActive: &inactive,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(list))
	})

	t.Run("SQL Escaping and Strict Sorting", func(t *testing.T) {
		p := "esc-" + time.Now().Format("150405")
		_ = repo.Create(ctx, p+"-100%", "http://100.com")
		_ = repo.Create(ctx, p+"-10_0", "http://10_0.com")
		_ = repo.Create(ctx, p+"-normal", "http://normal.com")

		// Search for "%" literal
		// Should match ONLY the link with % in its slug
		listPercent, _, err := repo.List(ctx, links.ListOptions{Keyword: "%"})
		require.NoError(t, err)

		foundPercent := false
		for _, l := range listPercent {
			if l.Slug == p+"-100%" {
				foundPercent = true
			}
			if l.Slug == p+"-normal" {
				t.Errorf("Should not have matched normal link when searching for '%%'")
			}
		}
		assert.True(t, foundPercent, "Should have found link with '%%'")

		// Search for "_" literal
		// Should match ONLY the link with _ in its slug
		listUnderscore, _, err := repo.List(ctx, links.ListOptions{Keyword: "_"})
		require.NoError(t, err)

		foundUnderscore := false
		for _, l := range listUnderscore {
			if l.Slug == p+"-10_0" {
				foundUnderscore = true
			}
			if l.Slug == p+"-normal" {
				t.Errorf("Should not have matched normal link when searching for '_'")
			}
		}
		assert.True(t, foundUnderscore, "Should have found link with '_'")

		// Strict Sorting Check
		// Create known sortable items
		_ = repo.Create(ctx, p+"-aaa", "http://aaa.com")
		_ = repo.Create(ctx, p+"-bbb", "http://bbb.com")
		_ = repo.Create(ctx, p+"-ccc", "http://ccc.com")

		listSort, _, err := repo.List(ctx, links.ListOptions{
			Keyword: p + "-",
			SortBy:  "slug",
			ListParams: request.ListParams{
				SortOrder: "ASC",
			},
		})
		require.NoError(t, err)

		// We expect at least the 3 new ones
		var sortedSlugs []string
		for _, l := range listSort {
			if strings.Contains(l.Slug, p+"-aaa") || strings.Contains(l.Slug, p+"-bbb") || strings.Contains(l.Slug, p+"-ccc") {
				sortedSlugs = append(sortedSlugs, l.Slug)
			}
		}

		require.Equal(t, 3, len(sortedSlugs))
		assert.Equal(t, p+"-aaa", sortedSlugs[0])
		assert.Equal(t, p+"-bbb", sortedSlugs[1])
		assert.Equal(t, p+"-ccc", sortedSlugs[2])
	})
}
