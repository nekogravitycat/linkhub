package tests

import (
	"context"
	"testing"
	"time"

	"github.com/nekogravitycat/linkhub/internal/links"
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

		list, err := repo.List(ctx, links.ListOptions{})
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(list), 2)

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
}
