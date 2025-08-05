package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nekogravitycat/linkhub/backend/internal/models"
	"github.com/nekogravitycat/linkhub/backend/internal/validator"
)

var ErrDuplicateSlug = errors.New("duplicate slug")

// GetResource retrieves a Resource by its slug.
// It fetches the Entry first, then loads either a Link or File based on the Entry's type.
func GetResource(ctx context.Context, slug string) (models.Resource, error) {
	entry, err := GetEntry(ctx, slug)
	if err != nil {
		return models.Resource{}, fmt.Errorf("failed to get entry: %w", err)
	}

	switch entry.Type {

	case models.ResourceTypeLink:
		link, err := GetLink(ctx, entry.ID)
		if err != nil {
			return models.Resource{}, fmt.Errorf("failed to get link: %w", err)
		}
		return models.Resource{
			Entry: entry,
			Link:  &link,
			File:  nil,
		}, nil

	case models.ResourceTypeFile:
		file, err := GetFile(ctx, entry.ID)
		if err != nil {
			return models.Resource{}, fmt.Errorf("failed to get file: %w", err)
		}
		return models.Resource{
			Entry: entry,
			Link:  nil,
			File:  &file,
		}, nil

	default:
		return models.Resource{}, fmt.Errorf("unknown resource type: %s", entry.Type)
	}
}

// InsertResource validates and inserts the given resource into the database.
// Entry ID will be omitted from the resource.
// Returns the entry ID of the inserted resource and an error if any.
// If the insertion fails, the returned entry ID will be -1.
func InsertResource(ctx context.Context, resource models.Resource) (int64, error) {
	if err := validator.ValidateResource(resource); err != nil {
		return -1, fmt.Errorf("invalid resource: %w", err)
	}

	var entryID int64 = -1

	db := GetDBClient()

	err := pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		const insertEntryQuery = `
			INSERT INTO entries (slug, type, password_hash, expires_at)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`
		err := tx.QueryRow(ctx, insertEntryQuery,
			resource.Entry.Slug,
			resource.Entry.Type,
			resource.Entry.PasswordHash,
			resource.Entry.ExpiresAt,
		).Scan(&entryID)
		if err != nil || entryID <= 0 {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" { // Unique violation
				return ErrDuplicateSlug
			}
			return fmt.Errorf("failed to insert entry: %w", err)
		}

		switch resource.Entry.Type {
		case models.ResourceTypeLink:
			const insertLinkQuery = `
				INSERT INTO links (entry_id, target_url)
				VALUES ($1, $2)
			`
			_, err = tx.Exec(ctx, insertLinkQuery, entryID, resource.Link.TargetURL)
			if err != nil {
				return fmt.Errorf("failed to insert link: %w", err)
			}

		case models.ResourceTypeFile:
			const insertFileQuery = `
				INSERT INTO files (entry_id, file_uuid, filename, mime_type, size)
				VALUES ($1, $2, $3, $4, $5)
			`
			_, err = tx.Exec(ctx, insertFileQuery,
				entryID,
				resource.File.FileUUID,
				resource.File.Filename,
				resource.File.MIMEType,
				resource.File.Size,
			)
			if err != nil {
				return fmt.Errorf("failed to insert file: %w", err)
			}

		default:
			return fmt.Errorf("unsupported resource type: %s", resource.Entry.Type)
		}

		return nil // commit will happen
	})

	if err != nil {
		return -1, fmt.Errorf("insert resource transaction failed: %w", err)
	}

	return entryID, nil
}

// UpdateResource validates and updates an existing resource in the database.
// - The entry type and created_at fields must not be changed.
// - If updatePassword is false, the existing password_hash will be preserved.
// - For link resources, only the target_url can be updated.
// - For file resources, the file content must remain the same.
// Returns an error if any validation or database operation fails.
func UpdateResource(ctx context.Context, resource models.Resource, updatePassword bool) error {
	if err := validator.ValidateResource(resource); err != nil {
		return fmt.Errorf("invalid resource: %w", err)
	}

	db := GetDBClient()

	err := pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		// Check if the existing entry type matches the resource type.
		var dbType string
		var dbPasswordHash *string
		var dbCreatedAt time.Time

		const selectEntryQuery = `
			SELECT type, created_at, password_hash
			FROM entries
			WHERE id = $1
		`
		if err := tx.QueryRow(ctx, selectEntryQuery, resource.Entry.ID).Scan(
			&dbType,
			&dbCreatedAt,
			&dbPasswordHash,
		); err != nil {
			return fmt.Errorf("failed to get entry type: %w", err)
		}

		// Check if the resource type matches.
		if models.ResourceType(dbType) != resource.Entry.Type {
			return fmt.Errorf("type cannot be updated")
		}

		// Check if the created_at timestamp matches.
		if !dbCreatedAt.UTC().Truncate(time.Second).Equal(resource.Entry.CreatedAt.UTC().Truncate(time.Second)) {
			return fmt.Errorf("created_at cannot be updated")
		}

		// Use the existing password hash if not updating.
		if !updatePassword {
			resource.Entry.PasswordHash = dbPasswordHash
		}

		// Update the entry.
		const updateEntryQuery = `
			UPDATE entries
			SET slug = $1, password_hash = $2, expires_at = $3
			WHERE id = $4
		`
		tag, err := tx.Exec(ctx, updateEntryQuery,
			resource.Entry.Slug,
			resource.Entry.PasswordHash,
			resource.Entry.ExpiresAt,
			resource.Entry.ID,
		)
		if err != nil {
			return fmt.Errorf("failed to update entry: %w", err)
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("entry update affected %d rows, expected 1", tag.RowsAffected())
		}

		switch resource.Entry.Type {
		case models.ResourceTypeLink:
			// If the resource is a link, update the link details.
			const updateLinkQuery = `
				UPDATE links
				SET target_url = $1
				WHERE entry_id = $2
			`
			tag, err := tx.Exec(ctx, updateLinkQuery, resource.Link.TargetURL, resource.Entry.ID)
			if err != nil {
				return fmt.Errorf("failed to update link: %w", err)
			}
			if tag.RowsAffected() != 1 {
				return fmt.Errorf("link update affected %d rows, expected 1", tag.RowsAffected())
			}

		case models.ResourceTypeFile:
			// Check if File is unchanged for file resources.
			var fileUUID, filename, mimeType string
			var size int64

			const selectFileQuery = `
				SELECT file_uuid, filename, mime_type, size
				FROM files
				WHERE entry_id = $1
			`
			if err := tx.QueryRow(ctx, selectFileQuery, resource.Entry.ID).Scan(
				&fileUUID,
				&filename,
				&mimeType,
				&size,
			); err != nil {
				return fmt.Errorf("failed to get file info: %w", err)
			}

			if fileUUID != resource.File.FileUUID ||
				filename != resource.File.Filename ||
				mimeType != resource.File.MIMEType ||
				size != resource.File.Size {
				return fmt.Errorf("file content changes are not supported")
			}
		default:
			return fmt.Errorf("unknown resource type: %s", resource.Entry.Type)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("update resource transaction failed: %w", err)
	}

	return nil
}
