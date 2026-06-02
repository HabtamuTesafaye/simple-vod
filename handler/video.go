package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"test_vod/config"
	"test_vod/s3"
	"test_vod/store"
)

type VideoHandler struct {
	store  store.Store
	s3Cli  *s3.Client
	config *config.Config
}

func NewVideoHandler(s store.Store, s3Cli *s3.Client, cfg *config.Config) *VideoHandler {
	return &VideoHandler{
		store:  s,
		s3Cli:  s3Cli,
		config: cfg,
	}
}

func (h *VideoHandler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.config.MaxUploadSize)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "File too large or invalid form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Missing file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	title := r.FormValue("title")
	if title == "" {
		title = header.Filename
	}

	folderIDStr := r.FormValue("folder_id")
	var folderID *string
	if folderIDStr != "" {
		folderID = &folderIDStr
	}

	// Basic type check
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(header.Filename), "."))
	allowed := false
	for _, a := range h.config.AllowedTypes {
		if ext == a {
			allowed = true
			break
		}
	}
	if !allowed {
		http.Error(w, "File type not allowed", http.StatusBadRequest)
		return
	}

	videoID := uuid.New().String()
	s3Key := fmt.Sprintf("videos/")
	if folderID != nil {
		s3Key += fmt.Sprintf("%s/", *folderID)
	} else {
		s3Key += "uncategorized/"
	}
	s3Key += fmt.Sprintf("%s.%s", videoID, ext)

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "video/mp4"
	}

	// 1. Upload to S3
	err = h.s3Cli.Upload(r.Context(), s3Key, file, contentType)
	if err != nil {
		http.Error(w, "Failed to upload to storage: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Save metadata
	video := &store.Video{
		ID:        videoID,
		FolderID:  folderID,
		Title:     title,
		Filename:  header.Filename,
		S3Key:     s3Key,
		MimeType:  contentType,
		SizeBytes: header.Size,
		Status:    store.StatusReady, // Skipping 'uploading' for this simple synchronous version
	}

	err = h.store.CreateVideo(r.Context(), video)
	if err != nil {
		// Rollback S3 on DB error
		_ = h.s3Cli.Delete(r.Context(), s3Key)
		http.Error(w, "Failed to save metadata", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(video)
}

func (h *VideoHandler) Stream(w http.ResponseWriter, r *http.Request) {
	videoID := chi.URLParam(r, "id")

	video, err := h.store.GetVideo(r.Context(), videoID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if video == nil {
		http.Error(w, "Video not found", http.StatusNotFound)
		return
	}

	url, err := h.s3Cli.PresignedURL(r.Context(), video.S3Key, h.config.PresignExpiry)
	if err != nil {
		http.Error(w, "Failed to generate presigned URL", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"video_id":   video.ID,
		"title":      video.Title,
		"duration":   video.Duration,
		"url":        url,
		"expires_in": int(h.config.PresignExpiry.Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *VideoHandler) List(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	folderIDStr := r.URL.Query().Get("folder_id")

	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset, _ := strconv.Atoi(offsetStr)
	if offset < 0 {
		offset = 0
	}

	var folderID *string
	if folderIDStr != "" {
		if folderIDStr != "null" && folderIDStr != "uncategorized" {
			folderID = &folderIDStr
		} // else folderID remains nil
	} else {
		// If no folder_id specified, you might want to list all, but for simplicity we match the schema behavior.
		// Actually let's interpret empty folder_id as "all" for this simple API if needed, 
		// but since the store expects a pointer for "null folder", let's adjust:
		// If folder_id parameter is completely missing, maybe return all.
		// To do "all" in our store, we'd need a separate method. For now, empty = root.
	}

	// Quick hack to list all if folder_id is not provided
	var videos []*store.Video
	var total int
	var err error

	if !r.URL.Query().Has("folder_id") {
		// We'd need to modify the store to support listing all. 
		// For now, let's just use the root folder (nil) as the default.
		videos, total, err = h.store.ListVideos(r.Context(), nil, limit, offset)
	} else {
		videos, total, err = h.store.ListVideos(r.Context(), folderID, limit, offset)
	}

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"data":  videos,
		"total": total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *VideoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	videoID := chi.URLParam(r, "id")

	video, err := h.store.GetVideo(r.Context(), videoID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if video == nil {
		http.Error(w, "Video not found", http.StatusNotFound)
		return
	}

	err = h.s3Cli.Delete(r.Context(), video.S3Key)
	if err != nil {
		// Log error, but continue to delete from DB to prevent orphaned DB records
		fmt.Printf("Warning: failed to delete from S3: %v\n", err)
	}

	err = h.store.DeleteVideo(r.Context(), videoID)
	if err != nil {
		http.Error(w, "Failed to delete metadata", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
