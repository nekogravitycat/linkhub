package database

import (
	"context"
	"fmt"

	"github.com/nekogravitycat/linkhub/backend/internal/models"
	"github.com/nekogravitycat/linkhub/backend/internal/validator"
)

func getLink(ctx context.Context, entryID int64) (models.Link, error) {
	if entryID <= 0 {
		return models.Link{}, fmt.Errorf("invalid entry ID: must be positive")
	}

	const query = `
		SELECT entry_id, target_url
		FROM links
		WHERE entry_id = $1
	`

	var link models.Link

	db := GetDBClient()

	row := db.QueryRow(ctx, query, entryID)
	if err := row.Scan(&link.EntryID, &link.TargetURL); err != nil {
		return models.Link{}, fmt.Errorf("failed to scan link: %w", err)
	}

	return link, nil
}

// UpdateLink updates the target URL of an existing link.
// Validates the link before updating.
func UpdateLink(ctx context.Context, link models.Link) error {
	if err := validator.ValidateLink(link); err != nil {
		return fmt.Errorf("failed to validate link: %w", err)
	}

	db := GetDBClient()

	const query = `
		UPDATE links
		SET target_url = $2
		WHERE entry_id = $1
	`
	cmdTag, err := db.Exec(ctx, query, link.EntryID, link.TargetURL)
	if err != nil {
		return fmt.Errorf("failed to update link: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("no link was updated (entry_id: %d)", link.EntryID)
	}

	return nil
}

// Use InsertResource to insert a link entry along with its link metadata.
// This ensures both the entry and the associated link are inserted atomically in a single transaction.

// Links are tied to entries and will be automatically deleted when the corresponding entry is deleted.
// Therefore, to delete a link, delete its associated entry instead.
