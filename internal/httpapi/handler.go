// Package httpapi implements the HTTP API for storing bucketed objects.
package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"bucket-object-store/internal/store"
)

const maxRequestBodyBytes = 10 << 20 // 10 MiB

type Handler struct {
	store *store.Store
}

func NewHandler(objectStore *store.Store) *Handler {
	return &Handler{store: objectStore}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bucketName, objectID, ok := parseObjectPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodPut:
		h.put(w, r, bucketName, objectID)
	case http.MethodGet:
		h.get(w, r, bucketName, objectID)
	case http.MethodDelete:
		h.delete(w, r, bucketName, objectID)
	default:
		w.Header().Set("Allow", "GET, PUT, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func parseObjectPath(path string) (bucketName, objectID string, ok bool) {
	const prefix = "/objects/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}

	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request, bucketName, objectID string) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBodyBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(w, "request body is larger than 10 MiB", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "could not read request body", http.StatusBadRequest)
		return
	}

	if err := h.store.Put(bucketName, objectID, body); err != nil {
		http.Error(w, "could not store object", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(struct {
		ID string `json:"id"`
	}{ID: objectID})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request, bucketName, objectID string) {
	body, exists := h.store.Get(bucketName, objectID)
	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, bucketName, objectID string) {
	if !h.store.Delete(bucketName, objectID) {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
}
