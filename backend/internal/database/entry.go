package database

import (
	"context"
	"fmt"

	"github.com/nekogravitycat/linkhub/backend/internal/models"
	"github.com/nekogravitycat/linkhub/backend/internal/validator"
)

func GetEntry(ctx context.Context, slug string) (models.Entry, error) {
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
		return models.Entry{}, fmt.Errorf("failed to scan entry: %w", err)
	}

	return entry, nil
}

func ListEntries(ctx context.Context) ([]models.Entry, error) {
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

// Use InsertResource to insert a entry along with its link/file metadata.
// This ensures both the entry and the associated link/file are inserted atomically in a single transaction.

/*
// Will omit entry.created_at since it will be set by the database
func InsertEntry(ctx context.Context, entry models.Entry) error {
	if err := validator.ValidateEntry(entry); err != nil {
		return err
	}

	db := GetDBClient()

	const query = `
		INSERT INTO entries (slug, type, password_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := db.Exec(ctx, query, entry.Slug, entry.Type, entry.PasswordHash, entry.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to insert entry: %w", err)
	}

	return nil
}
*/

// Will omit entry.created_at since it should not change
func UpdateEntry(ctx context.Context, entry models.Entry) error {
	if err := validator.ValidateEntry(entry); err != nil {
		return err
	}

	db := GetDBClient()

	const query = `
		UPDATE entries
		SET slug = $2, type = $3, password_hash = $4, expires_at = $5
		WHERE id = $1
	`
	_, err := db.Exec(ctx, query, entry.ID, entry.Slug, entry.Type, entry.PasswordHash, entry.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to update entry: %w", err)
	}

	return nil
}

// It deletes the entry and all associated resources (links/files).
// This is a cascading delete due to the ON DELETE CASCADE in the foreign key constraints.
func DeleteEntry(ctx context.Context, id int64) error {
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
		return fmt.Errorf("failed to delete entry: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("entry with id %d not found", id)
	}

	return nil
}
