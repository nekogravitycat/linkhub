package http

import (
	"errors"
	"regexp"
	"time"

	"github.com/nekogravitycat/linkhub/internal/pkg/request"
)

type BySlug struct {
	Slug string `uri:"slug" binding:"required"`
}

type CreateLinkRequest struct {
	Slug string `json:"slug" binding:"required"`
	URL  string `json:"url" binding:"required,url"`
}

type UpdateLinkRequest struct {
	URL      *string `json:"url" binding:"omitempty,url"`
	IsActive *bool   `json:"is_active"`
}

type ListRequest struct {
	request.ListParams
	SortBy string `form:"sort_by" binding:"omitempty,oneof=created_at updated_at slug id"`
}

type LinkResponse struct {
	ID        int64     `json:"id"`
	Slug      string    `json:"slug"`
	URL       string    `json:"url"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r *CreateLinkRequest) Validate() error {
	if r.URL == "" {
		return errors.New("url is required")
	}
	if r.Slug != "" {
		return ValidateSlug(r.Slug)
	}
	return nil
}

func (r *UpdateLinkRequest) Validate() error {
	if r.URL != nil && *r.URL == "" {
		return errors.New("url cannot be empty")
	}
	return nil
}

var slugRegex = regexp.MustCompile(`^[a-zA-Z0-9-_]+$`)

func ValidateSlug(slug string) error {
	if slug == "" {
		return errors.New("slug is required")
	}
	if len(slug) > 32 {
		return errors.New("slug is too long (max 32 chars)")
	}
	if !slugRegex.MatchString(slug) {
		return errors.New("slug contains invalid characters")
	}
	return nil
}
