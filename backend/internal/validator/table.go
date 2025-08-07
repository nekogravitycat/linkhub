package validator

import (
	"fmt"

	"github.com/nekogravitycat/linkhub/backend/internal/models"
)

// EntryID must be consistent across Resource and Link/File.
func ValidateResource(resource models.Resource) error {
	if err := ValidateEntry(resource.Entry); err != nil {
		return fmt.Errorf("invalid entry: %w", err)
	}

	switch resource.Entry.Type {
	case models.ResourceTypeLink:
		if resource.Link == nil {
			return fmt.Errorf("link details are required for link type")
		}
		if resource.File != nil {
			return fmt.Errorf("file details must not be set for link type")
		}
		if resource.Link.EntryID != resource.Entry.ID {
			return fmt.Errorf("entry ID mismatch between resource and link")
		}
		if err := ValidateLink(*resource.Link); err != nil {
			return fmt.Errorf("invalid link: %w", err)
		}

	case models.ResourceTypeFile:
		if resource.File == nil {
			return fmt.Errorf("file details are required for file type")
		}
		if resource.Link != nil {
			return fmt.Errorf("link details must not be set for file type")
		}
		if resource.File.EntryID != resource.Entry.ID {
			return fmt.Errorf("entry ID mismatch between resource and file")
		}
		if err := ValidateFile(*resource.File); err != nil {
			return fmt.Errorf("invalid file: %w", err)
		}

	default:
		return fmt.Errorf("unknown resource type: %s", resource.Entry.Type)
	}

	return nil
}

func ValidateEntry(entry models.Entry) error {
	if err := ValidateSlug(entry.Slug); err != nil {
		return fmt.Errorf("invalid slug: %w", err)
	}
	if err := ValidateType(entry.Type); err != nil {
		return fmt.Errorf("invalid resource type: %w", err)
	}
	if entry.PasswordHash != nil {
		if err := ValidatePasswordHash(*entry.PasswordHash); err != nil {
			return fmt.Errorf("invalid password hash: %w", err)
		}
	}
	return nil
}

func ValidateLink(link models.Link) error {
	if err := ValidateTargetURL(link.TargetURL); err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}
	return nil
}

func ValidateFile(file models.File) error {
	if err := ValidateUUID(file.FileUUID); err != nil {
		return fmt.Errorf("invalid file UUID: %w", err)
	}
	if err := ValidateFilename(file.Filename); err != nil {
		return fmt.Errorf("invalid filename: %w", err)
	}
	if err := ValidateMIMEType(file.MIMEType); err != nil {
		return fmt.Errorf("invalid MIME type: %w", err)
	}
	if err := ValidateSize(file.Size); err != nil {
		return fmt.Errorf("invalid file size: %w", err)
	}
	return nil
}

func ValidateEntryUpdate(fields models.EntryUpdate) error {
	if fields.Slug == nil && fields.PasswordHash == nil && fields.ExpiresAt == nil && !fields.UpdatePassword {
		return fmt.Errorf("at least one field must be provided for update")
	}
	if fields.Slug != nil {
		if err := ValidateSlug(*fields.Slug); err != nil {
			return fmt.Errorf("invalid slug: %w", err)
		}
	}
	if fields.PasswordHash != nil {
		if err := ValidatePasswordHash(*fields.PasswordHash); err != nil {
			return fmt.Errorf("invalid password hash: %w", err)
		}
	}
	// ExpiresAt should be validated in UpdateEntryRequest
	return nil
}
