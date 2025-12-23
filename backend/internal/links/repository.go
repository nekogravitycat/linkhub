package links

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrLinkNotFound = errors.New("link not found")
)

type Repository interface {
	Create(ctx context.Context, slug string, url string) error
	GetBySlug(ctx context.Context, slug string) (*Link, error)
	Update(ctx context.Context, link *Link) error
	Delete(ctx context.Context, slug string) error
	List(ctx context.Context, opts ListOptions) ([]*Link, error)
}

type repository struct {
	db *pgxpool.Pool
	sb sq.StatementBuilderType
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &repository{
		db: db,
		sb: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *repository) Create(ctx context.Context, slug string, url string) error {
	query := r.sb.Insert("links").
		Columns("slug", "url").
		Values(slug, url)

	sqlStr, args, err := query.ToSql()
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, sqlStr, args...)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) GetBySlug(ctx context.Context, slug string) (*Link, error) {
	query := r.sb.Select("id", "slug", "url", "is_active", "created_at", "updated_at").
		From("links").
		Where(sq.Eq{"slug": slug})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	var link Link

	err = r.db.QueryRow(ctx, sqlStr, args...).Scan(
		&link.ID,
		&link.Slug,
		&link.URL,
		&link.IsActive,
		&link.CreatedAt,
		&link.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLinkNotFound
		}
		return nil, err
	}

	return &link, nil
}

func (r *repository) Update(ctx context.Context, link *Link) error {
	query := r.sb.Update("links").
		Set("url", link.URL).
		Set("is_active", link.IsActive).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"slug": link.Slug})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		return err
	}

	result, err := r.db.Exec(ctx, sqlStr, args...)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrLinkNotFound
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, slug string) error {
	query := r.sb.Delete("links").
		Where(sq.Eq{"slug": slug})

	sqlStr, args, err := query.ToSql()
	if err != nil {
		return err
	}

	result, err := r.db.Exec(ctx, sqlStr, args...)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrLinkNotFound
	}

	return nil
}

func (r *repository) List(ctx context.Context, opts ListOptions) ([]*Link, error) {
	query := r.sb.Select("id", "slug", "url", "is_active", "created_at", "updated_at").
		From("links")

	if opts.IsActive != nil {
		query = query.Where(sq.Eq{"is_active": *opts.IsActive})
	}

	if opts.Keyword != "" {
		// Escape special characters for ILIKE
		escaper := strings.NewReplacer(
			`\`, `\\`,
			`%`, `\%`,
			`_`, `\_`,
		)
		cleanKeyword := escaper.Replace(opts.Keyword)
		pattern := "%" + cleanKeyword + "%"
		query = query.Where(sq.Or{
			sq.ILike{"slug": pattern},
			sq.ILike{"url": pattern},
		})
	}

	// Strict Sorting Validation
	sortMap := map[string]string{
		"created_at": "created_at",
		"updated_at": "updated_at",
		"slug":       "slug",
		"id":         "id",
	}

	sortByColumn, ok := sortMap[opts.SortBy]
	if !ok {
		sortByColumn = "created_at"
	}

	sortDirection := "DESC"
	if strings.ToUpper(opts.SortOrder) == "ASC" {
		sortDirection = "ASC"
	}

	query = query.OrderBy(fmt.Sprintf("%s %s", sortByColumn, sortDirection))

	if opts.Page > 0 && opts.PageSize > 0 {
		offset := uint64((opts.Page - 1) * opts.PageSize)
		query = query.Limit(uint64(opts.PageSize)).Offset(offset)
	}

	sqlStr, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*Link
	for rows.Next() {
		var link Link
		err := rows.Scan(
			&link.ID,
			&link.Slug,
			&link.URL,
			&link.IsActive,
			&link.CreatedAt,
			&link.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		links = append(links, &link)
	}

	return links, nil
}
