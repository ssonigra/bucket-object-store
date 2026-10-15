package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bucket-object-store/internal/store"
)

func TestPutGetAndDelete(t *testing.T) {
	handler := NewHandler(store.New())

	put := httptest.NewRequest(http.MethodPut, "/objects/photos/pic1", strings.NewReader("hello world"))
	putResponse := httptest.NewRecorder()
	handler.ServeHTTP(putResponse, put)
	if putResponse.Code != http.StatusCreated {
		t.Fatalf("PUT status = %d, want %d", putResponse.Code, http.StatusCreated)
	}
	var response struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(putResponse.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode PUT response: %v", err)
	}
	if response.ID != "pic1" {
		t.Fatalf("PUT response ID = %q, want %q", response.ID, "pic1")
	}

	get := httptest.NewRequest(http.MethodGet, "/objects/photos/pic1", nil)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, get)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", getResponse.Code, http.StatusOK)
	}
	if got := getResponse.Body.String(); got != "hello world" {
		t.Fatalf("GET body = %q, want %q", got, "hello world")
	}
	if got := getResponse.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("GET Content-Type = %q, want text/plain", got)
	}

	delete := httptest.NewRequest(http.MethodDelete, "/objects/photos/pic1", nil)
	deleteResponse := httptest.NewRecorder()
	handler.ServeHTTP(deleteResponse, delete)
	if deleteResponse.Code != http.StatusOK {
		t.Fatalf("DELETE status = %d, want %d", deleteResponse.Code, http.StatusOK)
	}

	missing := httptest.NewRequest(http.MethodGet, "/objects/photos/pic1", nil)
	missingResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingResponse, missing)
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("GET after DELETE status = %d, want %d", missingResponse.Code, http.StatusNotFound)
	}
}

func TestMissingObjectsAndInvalidPaths(t *testing.T) {
	handler := NewHandler(store.New())
	requests := []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/objects/photos/missing", http.StatusNotFound},
		{http.MethodDelete, "/objects/photos/missing", http.StatusNotFound},
		{http.MethodGet, "/objects/photos", http.StatusNotFound},
		{http.MethodGet, "/objects/photos/pic1/extra", http.StatusNotFound},
		{http.MethodPost, "/objects/photos/pic1", http.StatusMethodNotAllowed},
	}

	for _, tc := range requests {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			if response.Code != tc.want {
				t.Fatalf("status = %d, want %d", response.Code, tc.want)
			}
			if tc.want == http.StatusMethodNotAllowed && response.Header().Get("Allow") != "GET, PUT, DELETE" {
				t.Fatalf("Allow header = %q, want %q", response.Header().Get("Allow"), "GET, PUT, DELETE")
			}
		})
	}
}

func TestPutRejectsBodyOverLimit(t *testing.T) {
	handler := NewHandler(store.New())
	body := bytes.NewReader(make([]byte, maxRequestBodyBytes+1))
	request := httptest.NewRequest(http.MethodPut, "/objects/photos/pic1", body)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("PUT status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
}
