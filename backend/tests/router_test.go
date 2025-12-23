package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nekogravitycat/linkhub/internal/api"
	"github.com/nekogravitycat/linkhub/internal/config"
)

func TestNewRouter_CORS(t *testing.T) {
	tests := []struct {
		name          string
		isProduction  bool
		allowOrigins  []string
		requestOrigin string
		wantAllowed   bool
	}{
		{
			name:          "Development - Exact match",
			isProduction:  false,
			allowOrigins:  []string{"http://example.com"},
			requestOrigin: "http://example.com",
			wantAllowed:   true,
		},
		{
			name:          "Development - Localhost default port",
			isProduction:  false,
			allowOrigins:  []string{"http://example.com"},
			requestOrigin: "http://localhost",
			wantAllowed:   true,
		},
		{
			name:          "Development - Localhost custom port",
			isProduction:  false,
			allowOrigins:  []string{"http://example.com"},
			requestOrigin: "http://localhost:3000",
			wantAllowed:   true,
		},
		{
			name:          "Development - HTTPS Localhost",
			isProduction:  false,
			allowOrigins:  []string{"http://example.com"},
			requestOrigin: "https://localhost:8443",
			wantAllowed:   true,
		},
		{
			name:          "Development - Denied origin",
			isProduction:  false,
			allowOrigins:  []string{"http://example.com"},
			requestOrigin: "http://google.com",
			wantAllowed:   false,
		},
		{
			name:          "Production - Exact match",
			isProduction:  true,
			allowOrigins:  []string{"http://example.com"},
			requestOrigin: "http://example.com",
			wantAllowed:   true,
		},
		{
			name:          "Production - Localhost denied",
			isProduction:  true,
			allowOrigins:  []string{"http://example.com"},
			requestOrigin: "http://localhost:3000",
			wantAllowed:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				IsProduction: tt.isProduction,
				AllowOrigins: tt.allowOrigins,
			}

			// Pass nil for handler since we only test middleware
			// Method values from nil pointer are allowed in Go as long as they are not invoked
			router := api.NewRouter(cfg, nil)

			req := httptest.NewRequest(http.MethodOptions, "/links", nil)
			req.Host = "api.linkhub.com"
			req.Header.Set("Origin", tt.requestOrigin)
			req.Header.Set("Access-Control-Request-Method", "GET")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if tt.wantAllowed {
				if allowOrigin != tt.requestOrigin {
					t.Errorf("expected Access-Control-Allow-Origin to be %q, got %q", tt.requestOrigin, allowOrigin)
				}
			} else {
				if allowOrigin != "" {
					t.Errorf("expected Access-Control-Allow-Origin to be empty, got %q", allowOrigin)
				}
			}
		})
	}
}
