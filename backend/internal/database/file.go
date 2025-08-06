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

// Use InsertResource to insert a file entry along with its file metadata.
// This ensures both the entry and the associated file are inserted atomically in a single transaction.

// File updates are not supported.
// To update the file associated with a file entry, delete the entry and insert a new one.

// Files are tied to entries and will be automatically deleted when the corresponding entry is deleted.
// Therefore, to delete a file, delete its associated entry instead.
