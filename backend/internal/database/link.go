package database

import (
	"context"
	"fmt"

	"github.com/nekogravitycat/linkhub/backend/internal/models"
	"github.com/nekogravitycat/linkhub/backend/internal/validator"
)

func GetLink(ctx context.Context, entryID int64) (models.Link, error) {
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

func ListLinks(ctx context.Context) ([]models.Link, error) {
	const query = `
		SELECT entry_id, target_url
		FROM links
	`

	db := GetDBClient()

	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query links: %w", err)
	}
	defer rows.Close()

	var links []models.Link
	for rows.Next() {
		var link models.Link
		if err := rows.Scan(&link.EntryID, &link.TargetURL); err != nil {
			return nil, fmt.Errorf("failed to scan link: %w", err)
		}
		links = append(links, link)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate links: %w", err)
	}

	return links, nil
}

// Use InsertResource to insert a link entry along with its link metadata.
// This ensures both the entry and the associated link are inserted atomically in a single transaction.

/*
func InsertLink(ctx context.Context, link models.Link) error {
	if err := validator.ValidateLink(link); err != nil {
		return err
	}

	db := GetDBClient()

	const query = `
		INSERT INTO links (entry_id, target_url)
		VALUES ($1, $2)
	`
	_, err := db.Exec(ctx, query, link.EntryID, link.TargetURL)
	if err != nil {
		return fmt.Errorf("failed to insert link: %w", err)
	}

	return nil
}
*/

func UpdateLink(ctx context.Context, link models.Link) error {
	if err := validator.ValidateLink(link); err != nil {
		return err
	}

	db := GetDBClient()

	const query = `
		UPDATE links
		SET target_url = $1
		WHERE entry_id = $2
	`
	_, err := db.Exec(ctx, query, link.TargetURL, link.EntryID)
	if err != nil {
		return fmt.Errorf("failed to update link: %w", err)
	}

	return nil
}

// Links are tied to entries and will be automatically deleted when the corresponding entry is deleted.
// Therefore, to delete a link, delete its associated entry instead.
