package validator

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nekogravitycat/linkhub/backend/internal/models"
)

func ValidateRawSlug(slug string) error {
	if len(slug) < 1 {
		return fmt.Errorf("slug cannot be empty")
	}
	if len(slug) > 255 {
		return fmt.Errorf("slug cannot exceed 255 characters")
	}
	slugPattern := regexp.MustCompile(`^[A-Za-z0-9_\-]+$`)
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("slug can only contain alphanumeric characters, hyphens, and underscores")
	}
	return nil
}

func ValidateResourceType(resourceType string) error {
	validTypes := []string{"link", "file"}
	if !slices.Contains(validTypes, resourceType) {
		return fmt.Errorf("type invalid, must be one of %v", validTypes)
	}
	return nil
}

func ValidateTargetURL(targetURL *string) error {
	if targetURL == nil {
		return nil
	}
	if *targetURL == "" {
		return fmt.Errorf("target_url cannot be empty")
	}
	parsed, err := url.ParseRequestURI(*targetURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("target_url invalid")
	}
	return nil
}

func ValidateRawFilename(filename *string) error {
	if filename == nil {
		return nil
	}
	if len(*filename) < 1 {
		return fmt.Errorf("filename cannot be empty")
	}
	if len(*filename) > 255 {
		return fmt.Errorf("filename cannot exceed 255 characters")
	}
	if *filename == "." {
		return fmt.Errorf("filename cannot be '.'")
	}
	if strings.Contains(*filename, "..") {
		return fmt.Errorf("filename cannot contain consecutive dots")
	}
	if !utf8.ValidString(*filename) {
		return fmt.Errorf("filename must be a valid UTF-8 string")
	}
	illegalChars := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F\x7F]`)
	if illegalChars.MatchString(*filename) {
		return fmt.Errorf("filename contains invalid characters")
	}
	return nil
}

func ValidatePassword(password *string) error {
	if password == nil {
		return nil
	}
	if len(*password) < 1 {
		return fmt.Errorf("password cannot be empty")
	}
	if len(*password) > 255 {
		return fmt.Errorf("password cannot exceed 255 characters")
	}
	var validPassword = regexp.MustCompile(`^[A-Za-z0-9_!? \-]+$`)
	if !validPassword.MatchString(*password) {
		return fmt.Errorf("password can only contain letters, numbers, dash, underscore, exclamation, question mark, and space")
	}
	return nil
}

func ValidateExpiresAt(expiresAt *time.Time) error {
	if expiresAt == nil {
		return nil
	}
	if expiresAt.Before(time.Now().UTC()) {
		return fmt.Errorf("expires_at must be in the future")
	}
	return nil
}

func ValidateCreateResourceRequest(request models.CreateResourceRequest) error {
	if err := ValidateRawSlug(request.Slug); err != nil {
		return fmt.Errorf("slug validation failed: %w", err)
	}
	if err := ValidateResourceType(request.Type); err != nil {
		return fmt.Errorf("type validation failed: %w", err)
	}
	if err := ValidateTargetURL(request.TargetURL); err != nil {
		return fmt.Errorf("target_url validation failed: %w", err)
	}
	if err := ValidateRawFilename(request.Filename); err != nil {
		return fmt.Errorf("filename validation failed: %w", err)
	}
	if err := ValidatePassword(request.Password); err != nil {
		return fmt.Errorf("password validation failed: %w", err)
	}
	if err := ValidateExpiresAt(request.ExpiresAt); err != nil {
		return fmt.Errorf("expires_at validation failed: %w", err)
	}
	switch request.Type {
	case "link":
		if request.TargetURL == nil {
			return fmt.Errorf("target_url must be provided for link resources")
		}
		if request.Filename != nil {
			return fmt.Errorf("filename must not be provided for link resources")
		}
	case "file":
		if request.Filename == nil {
			return fmt.Errorf("filename must be provided for file resources")
		}
		if request.TargetURL != nil {
			return fmt.Errorf("target_url must not be provided for file resources")
		}
	default:
		return fmt.Errorf("invalid resource type: %s", request.Type)
	}
	return nil
}
