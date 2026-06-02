package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"test_vod/store"
)

type FolderHandler struct {
	store store.Store
}

func NewFolderHandler(s store.Store) *FolderHandler {
	return &FolderHandler{store: s}
}

type CreateFolderRequest struct {
	Name     string  `json:"name"`
	ParentID *string `json:"parent_id"`
}

func (h *FolderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	folder := &store.Folder{
		ID:       uuid.New().String(),
		Name:     req.Name,
		ParentID: req.ParentID,
	}

	err := h.store.CreateFolder(r.Context(), folder)
	if err != nil {
		http.Error(w, "Failed to create folder", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(folder)
}

func (h *FolderHandler) List(w http.ResponseWriter, r *http.Request) {
	parentIDStr := r.URL.Query().Get("parent_id")
	var parentID *string

	if parentIDStr != "" && parentIDStr != "null" {
		parentID = &parentIDStr
	}

	folders, err := h.store.ListFolders(r.Context(), parentID)
	if err != nil {
		http.Error(w, "Failed to fetch folders", http.StatusInternalServerError)
		return
	}

	// Always return an array, even if empty
	if folders == nil {
		folders = make([]*store.Folder, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(folders)
}

func (h *FolderHandler) Delete(w http.ResponseWriter, r *http.Request) {
	folderID := chi.URLParam(r, "id")

	// The store uses ON DELETE CASCADE for sub-folders, and ON DELETE SET NULL for videos
	err := h.store.DeleteFolder(r.Context(), folderID)
	if err != nil {
		http.Error(w, "Failed to delete folder", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
