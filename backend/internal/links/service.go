package links

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSlugTaken = errors.New("slug already taken")
)

type Service interface {
	Create(ctx context.Context, slug, url string) error
	Get(ctx context.Context, slug string) (*Link, error)
	List(ctx context.Context, opts ListOptions) ([]*Link, error)
	Update(ctx context.Context, slug string, url *string, isActive *bool) error
	Delete(ctx context.Context, slug string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) Create(ctx context.Context, slug, url string) error {
	// Check if slug exists
	_, err := s.repo.GetBySlug(ctx, slug)
	if err == nil {
		return ErrSlugTaken
	}
	if !errors.Is(err, ErrLinkNotFound) {
		return err
	}

	return s.repo.Create(ctx, slug, url)
}

func (s *service) Get(ctx context.Context, slug string) (*Link, error) {
	return s.repo.GetBySlug(ctx, slug)
}

func (s *service) List(ctx context.Context, opts ListOptions) ([]*Link, error) {
	return s.repo.List(ctx, opts)
}

func (s *service) Update(ctx context.Context, slug string, url *string, isActive *bool) error {
	link, err := s.repo.GetBySlug(ctx, slug)
	if err != nil {
		return err
	}

	if url != nil {
		link.URL = *url
	}
	if isActive != nil {
		link.IsActive = *isActive
	}
	link.UpdatedAt = time.Now()

	return s.repo.Update(ctx, link)
}

func (s *service) Delete(ctx context.Context, slug string) error {
	return s.repo.Delete(ctx, slug)
}
