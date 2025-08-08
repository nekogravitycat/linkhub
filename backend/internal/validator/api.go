package validator

import (
	"fmt"
	"time"

	"github.com/nekogravitycat/linkhub/backend/internal/models"
)

func ValidateUnlockResourceRequest(request models.UnlockResourceRequest) error {
	if err := ValidateRawPassword(request.Password); err != nil {
		return fmt.Errorf("invalid password: %w", err)
	}
	return nil
}

func ValidateUpdateEntryRequest(request models.UpdateEntryRequest, now time.Time) error {
	if request.UpdatePassword == nil {
		return fmt.Errorf("update_password field is required")
	}
	if request.RawSlug == nil && request.RawPassword == nil && request.ExpiresAt == nil && !*request.UpdatePassword {
		return fmt.Errorf("at least one field must be provided for update")
	}
	if request.RawSlug != nil {
		if err := ValidateRawSlug(*request.RawSlug); err != nil {
			return fmt.Errorf("invalid slug: %w", err)
		}
	}
	if request.RawPassword != nil {
		if err := ValidateRawPassword(*request.RawPassword); err != nil {
			return fmt.Errorf("invalid password: %w", err)
		}
	}
	if request.ExpiresAt != nil {
		if err := ValidateExpiresAt(*request.ExpiresAt, now); err != nil {
			return fmt.Errorf("invalid expiration date: %w", err)
		}
	}
	return nil
}

func ValidateCreateLinkRequest(request models.CreateLinkRequest, now time.Time) error {
	if err := ValidateRawSlug(request.RawSlug); err != nil {
		return fmt.Errorf("invalid slug: %w", err)
	}
	if err := ValidateTargetURL(request.TargetURL); err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}
	if request.RawPassword != nil {
		if err := ValidateRawPassword(*request.RawPassword); err != nil {
			return fmt.Errorf("invalid password: %w", err)
		}
	}
	if request.ExpiresAt != nil {
		if err := ValidateExpiresAt(*request.ExpiresAt, now); err != nil {
			return fmt.Errorf("invalid expiration date: %w", err)
		}
	}
	return nil
}

func ValidateUpdateLinkRequest(request models.UpdateLinkRequest) error {
	if err := ValidateTargetURL(request.TargetURL); err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}
	return nil
}

func ValidateCreateFileRequest(request models.CreateFileRequest, now time.Time) error {
	if err := ValidateRawSlug(request.RawSlug); err != nil {
		return fmt.Errorf("invalid slug: %w", err)
	}
	if err := ValidateFilename(request.Filename); err != nil {
		return fmt.Errorf("invalid filename: %w", err)
	}
	if err := ValidateMIMEType(request.MIMEType); err != nil {
		return fmt.Errorf("invalid MIME type: %w", err)
	}
	if err := ValidateSize(request.Size); err != nil {
		return fmt.Errorf("invalid file size: %w", err)
	}
	if request.RawPassword != nil {
		if err := ValidateRawPassword(*request.RawPassword); err != nil {
			return fmt.Errorf("invalid password: %w", err)
		}
	}
	if request.ExpiresAt != nil {
		if err := ValidateExpiresAt(*request.ExpiresAt, now); err != nil {
			return fmt.Errorf("invalid expiration date: %w", err)
		}
	}
	return nil
}

func ValidateUploadFileCompleteRequest(request models.UploadFileCompleteRequest) error {
	if err := ValidateUploadType(request.Type); err != nil {
		return fmt.Errorf("invalid upload type: %w", err)
	}
	if request.Type == models.UploadTypeMultipart {
		if request.Multipart == nil {
			return fmt.Errorf("multipart upload info is required for multipart uploads")
		}
		if err := ValidateMultipartCompleteInfo(*request.Multipart); err != nil {
			return fmt.Errorf("invalid multipart upload info: %w", err)
		}
	} else {
		if request.Multipart != nil {
			return fmt.Errorf("multipart upload info must not be set for single uploads")
		}
	}
	return nil
}

func ValidateMultipartCompleteInfo(request models.MultipartCompleteInfo) error {
	if err := ValidateUploadID(request.UploadID); err != nil {
		return fmt.Errorf("invalid upload ID: %w", err)
	}
	if len(request.Parts) == 0 {
		return fmt.Errorf("at least one part is required for multipart upload")
	}
	for idx, part := range request.Parts {
		if part.PartNumber != int32(idx+1) {
			return fmt.Errorf("part number must be sequential starting from 1, got %d at index %d", part.PartNumber, idx)
		}
		if err := ValidateMultipartCompletePart(part); err != nil {
			return fmt.Errorf("invalid multipart part: %w", err)
		}
	}
	return nil
}

func ValidateMultipartCompletePart(part models.MultipartCompletePart) error {
	if err := ValidatePartNumber(part.PartNumber); err != nil {
		return fmt.Errorf("invalid part number: %w", err)
	}
	if err := ValidateSingleFileETag(part.ETag); err != nil {
		return fmt.Errorf("invalid ETag: %w", err)
	}
	return nil
}

func ValidateS3HeadResponse(response models.S3HeadResponse) error {
	if err := ValidateMIMEType(response.MIMEType); err != nil {
		return fmt.Errorf("invalid MIME type: %w", err)
	}
	if response.Size <= 0 {
		return fmt.Errorf("invalid file size: must be positive")
	}
	return nil
}
