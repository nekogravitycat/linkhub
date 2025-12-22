package links

import (
	"time"

	"github.com/nekogravitycat/linkhub/internal/pkg/request"
)

type Link struct {
	ID        int64     `json:"id"`
	Slug      string    `json:"slug"`
	URL       string    `json:"url"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ListOptions struct {
	request.ListParams
	SortBy string
}
