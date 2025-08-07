package models

// RecourceType represents the type of resource, either a link or a file.
type ResourceType string

const (
	ResourceTypeLink ResourceType = "link"
	ResourceTypeFile ResourceType = "file"
)

// UploadType represents the type of upload, either single or multipart
type UploadType string

const (
	UploadTypeSingle    UploadType = "single"
	UploadTypeMultipart UploadType = "multipart"
)
