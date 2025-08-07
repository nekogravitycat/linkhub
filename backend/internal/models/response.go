package models

// GetResourceResponse defines the response structure when accessing a resource by slug.
type GetResourceResponse struct {
	Type ResourceType `json:"type" binding:"required"` // Type of the resource: "link" or "file"
	Link *PublicLink  `json:"link,omitempty"`          // Link resource data, present iff type == "link"
	File *PublicFile  `json:"file,omitempty"`          // File resource data, present iff type == "file"
}

// PublicLink represents the response payload for a link resource present to client.
type PublicLink struct {
	TargetURL string `json:"target_url"` // Destination URL to redirect when the link is accessed
}

// PublicFile represents the response payload for a file resource present to client.
type PublicFile struct {
	DownloadURL string `json:"download_url" binding:"required"` // Signed URL for downloading the file (from S3)
	Filename    string `json:"filename" binding:"required"`     // Original display name of the uploaded file
	MIMEType    string `json:"mime_type" binding:"required"`    // File MIME type (e.g., application/pdf, image/png)
	Size        int64  `json:"size" binding:"required"`         // File size in bytes
}

// UploadType represents the type of upload: single or multipart
type UploadType string

const (
	UploadTypeSingle    UploadType = "single"
	UploadTypeMultipart UploadType = "multipart"
)

// UploadFileResponse defines the response structure after requesting to upload a file
type UploadFileResponse struct {
	FileUUID  string                 `json:"file_uuid" binding:"required"` // UUID for the file, used as the S3 filename
	Type      UploadType             `json:"type" binding:"required"`      // Type of upload: "single" or "multipart"
	Single    *SingleUploadConfig    `json:"single,omitempty"`             // Config for single upload, present iff type == "single"
	Multipart *MultipartUploadConfig `json:"multipart,omitempty"`          // Config for multipart upload, present iff type == "multipart"
}

// SingleUploadConfig defines the configuration for a single file upload
type SingleUploadConfig struct {
	UploadURL string `json:"upload_url" binding:"required"` // Pre-signed URL for the single upload
}

// MultipartUploadConfig defines the configuration for a multipart file upload
type MultipartUploadConfig struct {
	UploadID string          `json:"upload_id" binding:"required"` // Multipart upload ID
	Parts    []MultipartPart `json:"parts" binding:"required"`     // List of all uploaded parts
}

// MultipartPart represents a single part in a multipart upload
type MultipartPart struct {
	PartNumber int    `json:"part_number" binding:"required"` // Part number (1-based index)
	UploadURL  string `json:"upload_url" binding:"required"`  // Pre-signed URL for this part
}
