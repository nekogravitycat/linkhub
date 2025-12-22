package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nekogravitycat/linkhub/internal/links"
	lhttp "github.com/nekogravitycat/linkhub/internal/links/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRouter() *gin.Engine {
	// Use pure Gin engine without default middleware for testing speed/clean logs,
	// or Default() if we want to test middlewares too. Default is safer for "server" tests.
	r := gin.Default()

	repo := links.NewRepository(testPool)
	svc := links.NewService(repo)
	handler := lhttp.NewHandler(svc)
	lhttp.RegisterRoutes(r, handler)

	return r
}

func TestHTTP_CreateLink(t *testing.T) {
	if testPool == nil {
		t.Skip("Postgres test pool not initialized")
	}

	r := setupRouter()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		slug := "http-create-" + time.Now().Format("150405000000")
		reqBody := map[string]string{
			"slug": slug,
			"url":  "https://example.com",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/private/links", bytes.NewBuffer(body))
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		// Verify DB
		_, err := links.NewRepository(testPool).GetBySlug(ctx, slug)
		assert.NoError(t, err)
	})

	t.Run("Duplicate Slug", func(t *testing.T) {
		slug := "http-dup-" + time.Now().Format("150405000000")
		// Setup existing
		_ = links.NewRepository(testPool).Create(ctx, slug, "https://1.com")

		reqBody := map[string]string{
			"slug": slug,
			"url":  "https://2.com",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/private/links", bytes.NewBuffer(body))
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("Invalid Payload", func(t *testing.T) {
		// Sending array instead of object
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/private/links", bytes.NewBufferString("[]"))
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid URL", func(t *testing.T) {
		reqBody := map[string]string{
			"slug": "valid-slug-bad-url-" + time.Now().Format("150405000000"),
			"url":  "not-a-valid-url",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/private/links", bytes.NewBuffer(body))
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHTTP_GetLink(t *testing.T) {
	if testPool == nil {
		t.Skip("Postgres test pool not initialized")
	}

	r := setupRouter()
	ctx := context.Background()
	repo := links.NewRepository(testPool)

	t.Run("Success", func(t *testing.T) {
		slug := "http-get-" + time.Now().Format("150405000000")
		_ = repo.Create(ctx, slug, "https://get.com")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/private/links/"+slug, nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Check Body...
	})

	t.Run("Not Found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/private/links/non-existent-slug", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHTTP_Redirect(t *testing.T) {
	if testPool == nil {
		t.Skip("Postgres test pool not initialized")
	}

	r := setupRouter()
	ctx := context.Background()
	repo := links.NewRepository(testPool)

	t.Run("Success", func(t *testing.T) {
		slug := "http-redir-" + time.Now().Format("150405000000")
		target := "https://example.org"
		_ = repo.Create(ctx, slug, target)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/public/"+slug, nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, target, w.Header().Get("Location"))
	})

	t.Run("Not Found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/public/nope", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Inactive Link", func(t *testing.T) {
		slug := "http-inactive-" + time.Now().Format("150405000000")
		// Manually create inactive link since repo create defaults to true
		// We use Update to set it inactive
		err := repo.Create(ctx, slug, "http://foo.com")
		require.NoError(t, err)

		link, err := repo.GetBySlug(ctx, slug)
		require.NoError(t, err)

		link.IsActive = false
		err = repo.Update(ctx, link)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/public/"+slug, nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code) // Handler returns 404 for inactive
	})
}

func TestHTTP_UpdateLink(t *testing.T) {
	if testPool == nil {
		t.Skip("Postgres test pool not initialized")
	}

	r := setupRouter()
	ctx := context.Background()
	repo := links.NewRepository(testPool)

	t.Run("Success Full Update", func(t *testing.T) {
		slug := "http-update-" + time.Now().Format("150405000000")
		_ = repo.Create(ctx, slug, "http://old.com")

		reqBody := map[string]interface{}{
			"url":       "http://new.com",
			"is_active": false,
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/private/links/"+slug, bytes.NewBuffer(body))
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify
		l, _ := repo.GetBySlug(ctx, slug)
		assert.Equal(t, "http://new.com", l.URL)
		assert.False(t, l.IsActive)
	})

	t.Run("Success Partial Update (IsActive Only)", func(t *testing.T) {
		slug := "http-partial-" + time.Now().Format("150405000000")
		_ = repo.Create(ctx, slug, "http://keep-me.com")

		// Only send is_active
		reqBody := map[string]any{
			"is_active": false,
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/private/links/"+slug, bytes.NewBuffer(body))
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify
		l, _ := repo.GetBySlug(ctx, slug)
		assert.Equal(t, "http://keep-me.com", l.URL) // URL should handle be unchanged
		assert.False(t, l.IsActive)
	})

	t.Run("Not Found", func(t *testing.T) {
		reqBody := map[string]any{
			"url":       "http://fail.com",
			"is_active": true,
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/private/links/phantom", bytes.NewBuffer(body))
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Invalid URL", func(t *testing.T) {
		slug := "http-update-bad-" + time.Now().Format("150405000000")
		_ = repo.Create(ctx, slug, "http://valid.com")

		reqBody := map[string]any{
			"url":       "not-a-valid-url",
			"is_active": true,
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/private/links/"+slug, bytes.NewBuffer(body))
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Empty URL in Update", func(t *testing.T) {
		slug := "http-update-empty-" + time.Now().Format("150405000000")
		_ = repo.Create(ctx, slug, "http://valid.com")

		// Sending empty string for URL should fail
		reqBody := map[string]any{
			"url": "",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PATCH", "/private/links/"+slug, bytes.NewBuffer(body))
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHTTP_DeleteLink(t *testing.T) {
	if testPool == nil {
		t.Skip("Postgres test pool not initialized")
	}

	r := setupRouter()
	ctx := context.Background()
	repo := links.NewRepository(testPool)

	t.Run("Success", func(t *testing.T) {
		slug := "http-del-" + time.Now().Format("150405000000")
		_ = repo.Create(ctx, slug, "http://bye.com")

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/private/links/"+slug, nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		_, err := repo.GetBySlug(ctx, slug)
		assert.Equal(t, links.ErrLinkNotFound, err)
	})

	t.Run("Not Found", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/private/links/ghost", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHTTP_ListLinks(t *testing.T) {
	if testPool == nil {
		t.Skip("Postgres test pool not initialized")
	}

	r := setupRouter()
	ctx := context.Background()
	repo := links.NewRepository(testPool)

	// Setup data for sorting/paging tests
	// Create links with specific Slugs to sort by
	// Use sleep to ensure createdAt difference if needed, but slug sorting is deterministic
	slugA := "a-slug-" + time.Now().Format("150405")
	slugB := "b-slug-" + time.Now().Format("150405")
	slugC := "c-slug-" + time.Now().Format("150405")

	_ = repo.Create(ctx, slugA, "https://a.com")
	_ = repo.Create(ctx, slugC, "https://c.com") // Created second, C
	_ = repo.Create(ctx, slugB, "https://b.com") // Created third, B

	// By default (created_at DESC): B, C, A

	t.Run("Default Sort (Created At Desc)", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/private/links", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var list []lhttp.LinkResponse
		err := json.Unmarshal(w.Body.Bytes(), &list)
		require.NoError(t, err)
		assert.NotEmpty(t, list)
		// Check that we find our slugs
		// Since other tests might add links, we filter to ours
		var filtered []lhttp.LinkResponse
		for _, l := range list {
			if l.Slug == slugA || l.Slug == slugB || l.Slug == slugC {
				filtered = append(filtered, l)
			}
		}
		// Expect B, C, A order (latest first)
		if len(filtered) >= 3 {
			assert.Equal(t, slugB, filtered[0].Slug)
			assert.Equal(t, slugC, filtered[1].Slug)
			assert.Equal(t, slugA, filtered[2].Slug)
		}
	})

	t.Run("Sort By Slug ASC", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/private/links?sort_by=slug&sort_order=asc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var list []lhttp.LinkResponse
		_ = json.Unmarshal(w.Body.Bytes(), &list)

		var filtered []lhttp.LinkResponse
		for _, l := range list {
			if l.Slug == slugA || l.Slug == slugB || l.Slug == slugC {
				filtered = append(filtered, l)
			}
		}

		if len(filtered) >= 3 {
			assert.Equal(t, slugA, filtered[0].Slug)
			assert.Equal(t, slugB, filtered[1].Slug)
			assert.Equal(t, slugC, filtered[2].Slug)
		}
	})

	t.Run("Pagination", func(t *testing.T) {
		// Just verify we get subset
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/private/links?page=1&page_size=1", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var list []lhttp.LinkResponse
		_ = json.Unmarshal(w.Body.Bytes(), &list)
		assert.Equal(t, 1, len(list))
	})

	t.Run("Invalid SortBy", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/private/links?sort_by=hacking", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid Page (Min)", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/private/links?page=0", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Invalid PageSize (Max)", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/private/links?page_size=101", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
