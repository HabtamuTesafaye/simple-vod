package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS folders (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		parent_id TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(parent_id) REFERENCES folders(id) ON DELETE CASCADE,
		UNIQUE(name, parent_id)
	);

	CREATE TABLE IF NOT EXISTS videos (
		id TEXT PRIMARY KEY,
		folder_id TEXT,
		title TEXT NOT NULL,
		filename TEXT NOT NULL,
		s3_key TEXT NOT NULL UNIQUE,
		mime_type TEXT NOT NULL,
		size_bytes INTEGER NOT NULL,
		duration INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(folder_id) REFERENCES folders(id) ON DELETE SET NULL
	);

	CREATE INDEX IF NOT EXISTS idx_videos_folder_id ON videos(folder_id);
	CREATE INDEX IF NOT EXISTS idx_folders_parent_id ON folders(parent_id);
	`
	_, err := s.db.Exec(query)
	return err
}

// -- Videos

func (s *SQLiteStore) CreateVideo(ctx context.Context, v *Video) error {
	v.CreatedAt = time.Now()
	v.UpdatedAt = time.Now()

	query := `
		INSERT INTO videos (id, folder_id, title, filename, s3_key, mime_type, size_bytes, duration, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		v.ID, v.FolderID, v.Title, v.Filename, v.S3Key, v.MimeType, v.SizeBytes, v.Duration, v.Status, v.CreatedAt, v.UpdatedAt)
	return err
}

func (s *SQLiteStore) GetVideo(ctx context.Context, id string) (*Video, error) {
	query := `
		SELECT id, folder_id, title, filename, s3_key, mime_type, size_bytes, duration, status, created_at, updated_at
		FROM videos WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)

	var v Video
	err := row.Scan(
		&v.ID, &v.FolderID, &v.Title, &v.Filename, &v.S3Key, &v.MimeType, &v.SizeBytes,
		&v.Duration, &v.Status, &v.CreatedAt, &v.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *SQLiteStore) UpdateVideo(ctx context.Context, v *Video) error {
	v.UpdatedAt = time.Now()
	query := `
		UPDATE videos
		SET folder_id = ?, title = ?, filename = ?, s3_key = ?, mime_type = ?, size_bytes = ?, duration = ?, status = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := s.db.ExecContext(ctx, query,
		v.FolderID, v.Title, v.Filename, v.S3Key, v.MimeType, v.SizeBytes, v.Duration, v.Status, v.UpdatedAt, v.ID)
	return err
}

func (s *SQLiteStore) DeleteVideo(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM videos WHERE id = ?", id)
	return err
}

func (s *SQLiteStore) ListVideos(ctx context.Context, folderID *string, limit, offset int) ([]*Video, int, error) {
	where := ""
	var args []interface{}

	if folderID != nil {
		where = "WHERE folder_id = ?"
		args = append(args, *folderID)
	} else {
		where = "WHERE folder_id IS NULL"
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM videos %s", where)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get page
	query := fmt.Sprintf(`
		SELECT id, folder_id, title, filename, s3_key, mime_type, size_bytes, duration, status, created_at, updated_at
		FROM videos %s ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, where)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var videos []*Video
	for rows.Next() {
		var v Video
		if err := rows.Scan(&v.ID, &v.FolderID, &v.Title, &v.Filename, &v.S3Key, &v.MimeType, &v.SizeBytes, &v.Duration, &v.Status, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, 0, err
		}
		videos = append(videos, &v)
	}
	return videos, total, nil
}

// -- Folders

func (s *SQLiteStore) CreateFolder(ctx context.Context, f *Folder) error {
	f.CreatedAt = time.Now()
	query := `INSERT INTO folders (id, name, parent_id, created_at) VALUES (?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, f.ID, f.Name, f.ParentID, f.CreatedAt)
	return err
}

func (s *SQLiteStore) GetFolder(ctx context.Context, id string) (*Folder, error) {
	query := `SELECT id, name, parent_id, created_at FROM folders WHERE id = ?`
	row := s.db.QueryRowContext(ctx, query, id)

	var f Folder
	err := row.Scan(&f.ID, &f.Name, &f.ParentID, &f.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *SQLiteStore) UpdateFolder(ctx context.Context, f *Folder) error {
	query := `UPDATE folders SET name = ?, parent_id = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, f.Name, f.ParentID, f.ID)
	return err
}

func (s *SQLiteStore) DeleteFolder(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM folders WHERE id = ?", id)
	return err
}

func (s *SQLiteStore) ListFolders(ctx context.Context, parentID *string) ([]*Folder, error) {
	where := ""
	var args []interface{}

	if parentID != nil {
		where = "WHERE parent_id = ?"
		args = append(args, *parentID)
	} else {
		where = "WHERE parent_id IS NULL"
	}

	query := fmt.Sprintf(`SELECT id, name, parent_id, created_at FROM folders %s ORDER BY name ASC`, where)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []*Folder
	for rows.Next() {
		var f Folder
		if err := rows.Scan(&f.ID, &f.Name, &f.ParentID, &f.CreatedAt); err != nil {
			return nil, err
		}
		folders = append(folders, &f)
	}
	return folders, nil
}
