package storage

import "time"

type Bucket struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Name      string    `json:"name"`
	Public    bool      `json:"public"`
	CreatedAt time.Time `json:"created_at"`
}

type Object struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	BucketID    string    `json:"bucket_id"`
	UserID      *string   `json:"user_id,omitempty"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	ContentType *string   `json:"content_type,omitempty"`
	ETag        *string   `json:"etag,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UploadURLRequest struct {
	Bucket      string `json:"bucket"`
	Path        string `json:"path"`
	ContentType string `json:"content_type"`
}

type UploadURLResponse struct {
	UploadURL string `json:"upload_url"`
	FileKey   string `json:"file_key"`
}

type ConfirmRequest struct {
	Bucket      string `json:"bucket"`
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	ETag        string `json:"etag"`
}

type DownloadURLRequest struct {
	Bucket string `json:"bucket"`
	Path   string `json:"path"`
}

type DownloadURLResponse struct {
	URL string `json:"url"`
}

type DeleteRequest struct {
	Bucket string `json:"bucket"`
	Path   string `json:"path"`
}
