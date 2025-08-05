package validator

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/nekogravitycat/linkhub/backend/internal/models"
)

// Validate if slug is 1 to 255 characters long.
// It may contain letters from any language (Unicode), digits, hyphens, and underscores.
func ValidateRawSlug(slug string) error {
	if len(slug) < 1 || len(slug) > 255 {
		return fmt.Errorf("slug must be between 1 and 255 characters long")
	}
	slugPattern := regexp.MustCompile(`^[\p{L}\d_-]+$`)
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("slug can only contain alphanumeric characters, hyphens, and underscores")
	}
	return nil
}

// Validate if slug is 1 to 765 characters long.
// It must be a canonical URL-encoded string.
func ValidateSlug(slug string) error {
	if len(slug) < 1 || len(slug) > 765 {
		return fmt.Errorf("slug must be between 1 and 765 characters long")
	}
	decoded, err := url.PathUnescape(slug)
	if err != nil {
		return fmt.Errorf("slug must be a valid URL encoded string")
	}
	reEncoded := url.PathEscape(decoded)
	if reEncoded != slug {
		return fmt.Errorf("slug must be a canonical URL encoded string: %s -> %s", slug, reEncoded)
	}
	return nil
}

func ValidateType(resourceType models.ResourceType) error {
	if resourceType != models.ResourceTypeLink && resourceType != models.ResourceTypeFile {
		return fmt.Errorf("invalid resource type: %s", resourceType)
	}
	return nil
}

// Validate if password is 1 to 255 characters long.
// It must contain only letters, numbers, dash, underscore, exclamation mark, question mark, and space.
func ValidateRawPassword(password string) error {
	if len(password) < 1 || len(password) > 255 {
		return fmt.Errorf("password must be between 1 and 255 characters long")
	}
	var validPassword = regexp.MustCompile(`^[A-Za-z0-9_!? \-]+$`)
	if !validPassword.MatchString(password) {
		return fmt.Errorf("password can only contain letters, numbers, dash, underscore, exclamation, question mark, and space")
	}
	return nil
}

// Validate if password_hash is a valid bcrypt 2 hash.
func ValidatePasswordHash(passwordHash string) error {
	var bcrypt2Pattern = regexp.MustCompile(`^\$2[abxy]\$\d{2}\$[./A-Za-z0-9]{53}$`)
	if !bcrypt2Pattern.MatchString(passwordHash) {
		return fmt.Errorf("invalid bcrypt $2$ hash format")
	}
	return nil
}

// Validate if target_url is 1 to 2000 characters long.
// It must be a valid URL with scheme and host.
func ValidateTargetURL(targetURL string) error {
	if len(targetURL) < 1 || len(targetURL) > 2000 {
		return fmt.Errorf("target_url must be between 1 and 2000 characters long")
	}
	parsed, err := url.ParseRequestURI(targetURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("target_url must be a valid URL with scheme and host")
	}
	return nil
}

// Validate if the input string is a valid UUID version 4.
func ValidateUUID(u string) error {
	parsed, err := uuid.Parse(u)
	if err != nil {
		return fmt.Errorf("invalid UUID v4 format: %w", err)
	}
	if parsed.Version() != 4 {
		return fmt.Errorf("UUID must be version 4")
	}
	return nil
}

// Validate if filename is 1 to 255 characters long.
// It must be valid UTF-8 and not contain illegal characters such as <, >, :, ", /,\, |, ?, *, or control characters.
func ValidateFilename(filename string) error {
	if len(filename) < 1 || len(filename) > 255 {
		return fmt.Errorf("filename must be between 1 and 255 characters long")
	}
	if filename == "." {
		return fmt.Errorf("filename cannot be '.'")
	}
	if strings.Contains(filename, "..") {
		return fmt.Errorf("filename cannot contain consecutive dots")
	}
	if !utf8.ValidString(filename) {
		return fmt.Errorf("filename must be a valid UTF-8 string")
	}
	illegalChars := regexp.MustCompile(`[<>:"/\\|?*\p{C}]`)
	if illegalChars.MatchString(filename) {
		return fmt.Errorf("filename contains invalid characters")
	}
	return nil
}

// Validate if mime_type is 1 to 127 characters long.
// It must be in the format "type/subtype".
func ValidateMIMEType(mimeType string) error {
	if len(mimeType) < 1 || len(mimeType) > 127 {
		return fmt.Errorf("mime_type must be between 1 and 127 characters long")
	}
	mimeParts := strings.SplitN(mimeType, "/", 2)
	if len(mimeParts) != 2 || mimeParts[0] == "" || mimeParts[1] == "" {
		return fmt.Errorf("mime_type must be in the format 'type/subtype'")
	}
	return nil
}

// Validate if size is between 0 and 10 GB.
// It must be a non-negative integer.
func ValidateSize(size int64) error {
	if size <= 0 {
		return fmt.Errorf("size must be a positive integer")
	}
	if size > 10*1024*1024*1024 { // 10 GB limit
		return fmt.Errorf("size cannot exceed 10 GB")
	}
	return nil
}

// Validate if expires_at is a future timestamp.
// Input must be in UTC in order to validate correctly.
func ValidateExpiresAt(expiresAt time.Time, now time.Time) error {
	if expiresAt.UTC().Before(now.UTC()) {
		return fmt.Errorf("expires_at must be in the future")
	}
	return nil
}
