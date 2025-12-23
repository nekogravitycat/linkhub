package tests

import (
	"strings"
	"testing"

	lhttp "github.com/nekogravitycat/linkhub/internal/links/http"
)

func ptrString(s string) *string { return &s }
func ptrBool(b bool) *bool       { return &b }

func TestValidateSlug(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{"valid lowercase", "valid-slug", false},
		{"valid uppercase", "VALID-SLUG", false},
		{"valid mixed", "Valid-Slug-123", false},
		{"valid underscore", "valid_slug", false},
		{"empty", "", true},
		{"invalid char space", "invalid slug", true},
		{"invalid char special", "invalid@slug", true},
		{"invalid char slash", "invalid/slug", true},
		{"too long", strings.Repeat("a", 33), true},
		{"max length", strings.Repeat("a", 32), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := lhttp.ValidateSlug(tt.slug)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSlug() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateLinkRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     lhttp.CreateLinkRequest
		wantErr bool
	}{
		{
			name: "valid",
			req: lhttp.CreateLinkRequest{
				Slug: "valid-slug",
				URL:  "https://example.com",
			},
			wantErr: false,
		},
		{
			name: "valid no slug",
			req: lhttp.CreateLinkRequest{
				Slug: "",
				URL:  "https://example.com",
			},
			wantErr: false,
		},
		{
			name: "missing url",
			req: lhttp.CreateLinkRequest{
				Slug: "valid-slug",
				URL:  "",
			},
			wantErr: true,
		},
		{
			name: "invalid slug",
			req: lhttp.CreateLinkRequest{
				Slug: "invalid slug",
				URL:  "https://example.com",
			},
			wantErr: true,
		},
		{
			name: "url too long",
			req: lhttp.CreateLinkRequest{
				Slug: "valid-slug",
				URL:  "https://example.com/" + strings.Repeat("a", 2048),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateLinkRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateLinkRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     lhttp.UpdateLinkRequest
		wantErr bool
	}{
		{
			name: "valid",
			req: lhttp.UpdateLinkRequest{
				URL:      ptrString("https://example.com"),
				IsActive: ptrBool(true),
			},
			wantErr: false,
		},
		{
			name: "valid partial (no url)",
			req: lhttp.UpdateLinkRequest{
				URL:      nil,
				IsActive: ptrBool(true),
			},
			wantErr: false,
		},
		{
			name: "empty url",
			req: lhttp.UpdateLinkRequest{
				URL:      ptrString(""),
				IsActive: ptrBool(true),
			},
			wantErr: true,
		},
		{
			name: "url too long",
			req: lhttp.UpdateLinkRequest{
				URL:      ptrString("https://example.com/" + strings.Repeat("a", 2048)),
				IsActive: ptrBool(true),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateLinkRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     lhttp.ListRequest
		wantErr bool
	}{
		{
			name: "valid keyword",
			req: lhttp.ListRequest{
				Keyword: "valid",
			},
			wantErr: false,
		},
		{
			name: "valid keyword with spaces",
			req: lhttp.ListRequest{
				Keyword: "  valid  ",
			},
			wantErr: false,
		},
		{
			name: "keyword too short",
			req: lhttp.ListRequest{
				Keyword: "hi",
			},
			wantErr: true,
		},
		{
			name: "keyword too short after trim",
			req: lhttp.ListRequest{
				Keyword: "  hi  ",
			},
			wantErr: true,
		},
		{
			name: "keyword too long",
			req: lhttp.ListRequest{
				Keyword: strings.Repeat("a", 65),
			},
			wantErr: true,
		},
		{
			name: "keyword with control chars",
			req: lhttp.ListRequest{
				Keyword: "val\x00id",
			},
			wantErr: false, // Should be sanitized to "valid" (5 chars)
		},
		{
			name: "keyword short after sanitization",
			req: lhttp.ListRequest{
				Keyword: "ab\x00",
			},
			wantErr: true, // "ab" < 3 chars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We work on a copy to test side effects (sanitization)
			req := tt.req
			err := req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ListRequest.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify sanitization
			if !tt.wantErr && strings.Contains(req.Keyword, "\x00") {
				t.Errorf("ListRequest.Validate() did not sanitize control characters")
			}
			if !tt.wantErr && strings.TrimSpace(req.Keyword) != req.Keyword {
				t.Errorf("ListRequest.Validate() did not trim spaces")
			}
		})
	}
}
