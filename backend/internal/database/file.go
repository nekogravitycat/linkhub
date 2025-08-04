package database

import (
	"context"
	"fmt"

	"github.com/nekogravitycat/linkhub/backend/internal/models"
)

func GetFile(ctx context.Context, entryID int64) (models.File, error) {
	if entryID <= 0 {
		return models.File{}, fmt.Errorf("invalid entry ID: must be positive")
	}

	const query = `
		SELECT entry_id, file_uuid, filename, mime_type, size
		FROM files
		WHERE entry_id = $1
	`

	var file models.File

	db := GetDBClient()

	row := db.QueryRow(ctx, query, entryID)
	if err := row.Scan(&file.EntryID, &file.FileUUID, &file.Filename, &file.MIMEType, &file.Size); err != nil {
		return models.File{}, fmt.Errorf("failed to scan file: %w", err)
	}

	return file, nil
}

func ListFiles(ctx context.Context) ([]models.File, error) {
	const query = `
		SELECT entry_id, file_uuid, filename, mime_type, size
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

/*
func InsertFile(ctx context.Context, file models.File) error {
	if err := validator.ValidateFile(file); err != nil {
		return err
	}

	db := GetDBClient()

	const query = `
		INSERT INTO files (entry_id, file_uuid, filename, mime_type, size)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := db.Exec(ctx, query, file.EntryID, file.FileUUID, file.Filename, file.MIMEType, file.Size)
	if err != nil {
		return fmt.Errorf("failed to insert file: %w", err)
	}

	return nil
}
*/

// File updates are not supported.
// To update the file associated with a file entry, delete the entry and insert a new one.

/*
func UpdateFile(ctx context.Context, file models.File) error {
	if err := validator.ValidateFile(file); err != nil {
		return err
	}

	db := GetDBClient()

	const query = `
		UPDATE files
		SET file_uuid = $2, filename = $3, mime_type = $4, size = $5
		WHERE entry_id = $1
	`
	_, err := db.Exec(ctx, query, file.EntryID, file.FileUUID, file.Filename, file.MIMEType, file.Size)
	if err != nil {
		return fmt.Errorf("failed to update file: %w", err)
	}

	return nil
}
*/

// Files are tied to entries and will be automatically deleted when the corresponding entry is deleted.
// Therefore, to delete a file, delete its associated entry instead.
