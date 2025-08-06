package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nekogravitycat/linkhub/backend/internal/models"
	"github.com/nekogravitycat/linkhub/backend/internal/validator"
)

func getEntry(ctx context.Context, slug string) (models.Entry, error) {
	if err := validator.ValidateSlug(slug); err != nil {
		return models.Entry{}, err
	}

	const query = `
		SELECT id, slug, type, password_hash, created_at, expires_at
		FROM entries
		WHERE slug = $1
	`

	var entry models.Entry

	db := GetDBClient()

	row := db.QueryRow(ctx, query, slug)
	if err := row.Scan(
		&entry.ID,
		&entry.Slug,
		&entry.Type,
		&entry.PasswordHash,
		&entry.CreatedAt,
		&entry.ExpiresAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Entry{}, ErrEntryNotFound
		}
		return models.Entry{}, fmt.Errorf("failed to scan entry: %w", err)
	}

	return entry, nil
}

func listEntries(ctx context.Context) ([]models.Entry, error) {
	const query = `
		SELECT id, slug, type, password_hash, created_at, expires_at
		FROM entries
	`
	db := GetDBClient()

	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}
	defer rows.Close()

	var entries []models.Entry
	for rows.Next() {
		var entry models.Entry
		if err := rows.Scan(
			&entry.ID,
			&entry.Slug,
			&entry.Type,
			&entry.PasswordHash,
			&entry.CreatedAt,
			&entry.ExpiresAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate entries: %w", err)
	}

	return entries, nil
}

// Update slug, password_hash, and expires_at of an entry.
// If updatePassword is false, it uses the existing password hash.
func UpdateEntry(ctx context.Context, oldSlug string, fields models.EntryUpdate) error {
	if err := validator.ValidateEntryUpdate(fields); err != nil {
		return fmt.Errorf("failed to validate entry update: %w", err)
	}

	// Get the existing entry
	existingEntry, err := getEntry(ctx, oldSlug)
	if err != nil {
		if errors.Is(err, ErrEntryNotFound) {
			return fmt.Errorf("entry not found: %w", err)
		}
		return fmt.Errorf("failed to get existing entry: %w", err)
	}

	// Use the existing slug if not provided
	if fields.Slug == nil {
		fields.Slug = &existingEntry.Slug
	}
	// Use the existing expires_at if not provided
	if fields.ExpiresAt == nil {
		fields.ExpiresAt = existingEntry.ExpiresAt
	}
	// Use the existing password hash if not updating
	if !fields.UpdatePassword {
		fields.PasswordHash = existingEntry.PasswordHash
	}

	db := GetDBClient()

	// Update the entry in the database
	const query = `
		UPDATE entries
		SET slug = $2, password_hash = $3, expires_at = $4
		WHERE id = $1
	`
	cmdTag, err := db.Exec(ctx, query,
		existingEntry.ID,
		fields.Slug,
		fields.PasswordHash,
		fields.ExpiresAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // Unique violation
			return ErrDuplicateSlug
		}
		return fmt.Errorf("failed to update entry: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("no entry was updated (slug: %s)", oldSlug)
	}

	return nil
}

// It deletes the entry and all associated resources (links/files).
// This is a cascading delete due to the ON DELETE CASCADE in the foreign key constraints.
func deleteEntry(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid entry ID: must be positive")
	}

	const query = `
		DELETE FROM entries
		WHERE id = $1
	`

	db := GetDBClient()

	cmdTag, err := db.Exec(ctx, query, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEntryNotFound
		}
		return fmt.Errorf("failed to delete entry: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("entry with id %d not found", id)
	}

	return nil
}
