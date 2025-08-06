package database

import (
	"context"
	"fmt"

	"github.com/nekogravitycat/linkhub/backend/internal/models"
)

func getFile(ctx context.Context, entryID int64) (models.File, error) {
	if entryID <= 0 {
		return models.File{}, fmt.Errorf("invalid entry ID: must be positive")
	}

	const query = `
		SELECT entry_id, file_uuid, filename, mime_type, size, pending
		FROM files
		WHERE entry_id = $1
	`

	var file models.File

	db := GetDBClient()

	row := db.QueryRow(ctx, query, entryID)
	if err := row.Scan(
		&file.EntryID,
		&file.FileUUID,
		&file.Filename,
		&file.MIMEType,
		&file.Size,
		&file.Pending,
	); err != nil {
		return models.File{}, fmt.Errorf("failed to scan file: %w", err)
	}

	return file, nil
}

func listFiles(ctx context.Context) ([]models.File, error) {
	const query = `
		SELECT entry_id, file_uuid, filename, mime_type, size, pending
		FROM files
	`

	db := GetDBClient()

	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query files: %w", err)
	}
	defer rows.Close()

	var files []models.File
	for rows.Next() {
		var file models.File
		if err := rows.Scan(
			&file.EntryID,
			&file.FileUUID,
			&file.Filename,
			&file.MIMEType,
			&file.Size,
			&file.Pending,
		); err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		files = append(files, file)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate files: %w", err)
	}

	return files, nil
}

// Use InsertResource to insert a file entry along with its file metadata.
// This ensures both the entry and the associated file are inserted atomically in a single transaction.

// File updates are not supported.
// To update the file associated with a file entry, delete the entry and insert a new one.

// Files are tied to entries and will be automatically deleted when the corresponding entry is deleted.
// Therefore, to delete a file, delete its associated entry instead.
