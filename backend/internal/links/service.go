package links

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrSlugTaken    = errors.New("slug already taken")
	ErrRedirectLoop = errors.New("target url cannot contain redirect domain")
)

type Service interface {
	Create(ctx context.Context, slug, url string) error
	Get(ctx context.Context, slug string) (*Link, error)
	List(ctx context.Context, opts ListOptions) ([]*Link, int64, error)
	Update(ctx context.Context, slug string, url *string, isActive *bool) error
	Delete(ctx context.Context, slug string) error
}

type service struct {
	repo           Repository
	redirectDomain string
}

func NewService(repo Repository, redirectDomain string) Service {
	return &service{
		repo:           repo,
		redirectDomain: redirectDomain,
	}
}

func (s *service) Create(ctx context.Context, slug, url string) error {
	if strings.Contains(url, s.redirectDomain) {
		return ErrRedirectLoop
	}

	// The repository maps the slug UNIQUE-constraint violation to ErrSlugTaken,
	// so no separate existence check is needed (one round-trip instead of two).
	return s.repo.Create(ctx, slug, url)
}

func (s *service) Get(ctx context.Context, slug string) (*Link, error) {
	return s.repo.GetBySlug(ctx, slug)
}

func (s *service) List(ctx context.Context, opts ListOptions) ([]*Link, int64, error) {
	return s.repo.List(ctx, opts)
}

func (s *service) Update(ctx context.Context, slug string, url *string, isActive *bool) error {
	link, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return err
	}

	if url != nil {
		if strings.Contains(*url, s.redirectDomain) {
			return ErrRedirectLoop
		}
		link.URL = *url
	}
	if isActive != nil {
		link.IsActive = *isActive
	}
	// updated_at is maintained by the DB trigger; no need to set it here.

	return s.repo.Update(ctx, link)
}

func (s *service) Delete(ctx context.Context, slug string) error {
	return s.repo.Delete(ctx, slug)
}
