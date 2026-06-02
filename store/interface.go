package store

import (
	"context"
	"time"
)

type VideoStatus string

const (
	StatusUploading VideoStatus = "uploading"
	StatusReady     VideoStatus = "ready"
	StatusFailed    VideoStatus = "failed"
)

type Video struct {
	ID        string      `json:"id"`
	FolderID  *string     `json:"folder_id,omitempty"`
	Title     string      `json:"title"`
	Filename  string      `json:"filename"`
	S3Key     string      `json:"s3_key"`
	MimeType  string      `json:"mime_type"`
	SizeBytes int64       `json:"size_bytes"`
	Duration  int         `json:"duration"`
	Status    VideoStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type Folder struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ParentID  *string   `json:"parent_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Store interface {
	// Videos
	CreateVideo(ctx context.Context, v *Video) error
	GetVideo(ctx context.Context, id string) (*Video, error)
	UpdateVideo(ctx context.Context, v *Video) error
	DeleteVideo(ctx context.Context, id string) error
	ListVideos(ctx context.Context, folderID *string, limit, offset int) ([]*Video, int, error)

	// Folders
	CreateFolder(ctx context.Context, f *Folder) error
	GetFolder(ctx context.Context, id string) (*Folder, error)
	UpdateFolder(ctx context.Context, f *Folder) error
	DeleteFolder(ctx context.Context, id string) error
	ListFolders(ctx context.Context, parentID *string) ([]*Folder, error)

	Close() error
}
