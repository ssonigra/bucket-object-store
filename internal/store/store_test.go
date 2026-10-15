package store

import (
	"bytes"
	"testing"
)

func TestDeduplicatesWithinEachBucket(t *testing.T) {
	s := New()
	content := []byte("same text")

	for _, objectID := range []string{"pic1", "pic2"} {
		if err := s.Put("photos", objectID, content); err != nil {
			t.Fatalf("Put(photos, %s): %v", objectID, err)
		}
	}
	if err := s.Put("archive", "copy1", content); err != nil {
		t.Fatalf("Put(archive, copy1): %v", err)
	}

	photos := s.buckets["photos"]
	archive := s.buckets["archive"]
	if got := len(photos.blobs); got != 1 {
		t.Fatalf("photos has %d stored copies, want 1", got)
	}
	if got := len(archive.blobs); got != 1 {
		t.Fatalf("archive has %d stored copies, want 1", got)
	}
	if got := photos.refs[photos.objects["pic1"]]; got != 2 {
		t.Fatalf("photos copy has %d references, want 2", got)
	}
	if got := archive.refs[archive.objects["copy1"]]; got != 1 {
		t.Fatalf("archive copy has %d references, want 1", got)
	}

	photoHash := photos.objects["pic1"]
	archiveHash := archive.objects["copy1"]
	if &photos.blobs[photoHash][0] == &archive.blobs[archiveHash][0] {
		t.Fatal("different buckets unexpectedly share the same stored byte slice")
	}
}

func TestDeleteKeepsSharedContentUntilLastObjectIsRemoved(t *testing.T) {
	s := New()
	content := []byte("shared text")
	if err := s.Put("photos", "pic1", content); err != nil {
		t.Fatal(err)
	}
	if err := s.Put("photos", "pic2", content); err != nil {
		t.Fatal(err)
	}

	if !s.Delete("photos", "pic1") {
		t.Fatal("Delete(pic1) returned false, want true")
	}
	got, exists := s.Get("photos", "pic2")
	if !exists || !bytes.Equal(got, content) {
		t.Fatalf("Get(pic2) = %q, %t; want %q, true", got, exists, content)
	}
	if len(s.buckets["photos"].blobs) != 1 {
		t.Fatal("shared content was removed while pic2 still referred to it")
	}

	if !s.Delete("photos", "pic2") {
		t.Fatal("Delete(pic2) returned false, want true")
	}
	if _, exists := s.buckets["photos"]; exists {
		t.Fatal("empty bucket was not removed")
	}
}

func TestPutReplacesObjectAndGetReturnsCopy(t *testing.T) {
	s := New()
	if err := s.Put("photos", "pic1", []byte("old text")); err != nil {
		t.Fatal(err)
	}
	if err := s.Put("photos", "pic2", []byte("old text")); err != nil {
		t.Fatal(err)
	}
	if err := s.Put("photos", "pic1", []byte("new text")); err != nil {
		t.Fatal(err)
	}

	newText, exists := s.Get("photos", "pic1")
	if !exists || string(newText) != "new text" {
		t.Fatalf("Get(pic1) = %q, %t; want %q, true", newText, exists, "new text")
	}
	oldText, exists := s.Get("photos", "pic2")
	if !exists || string(oldText) != "old text" {
		t.Fatalf("Get(pic2) = %q, %t; want %q, true", oldText, exists, "old text")
	}

	newText[0] = 'X'
	unchanged, _ := s.Get("photos", "pic1")
	if string(unchanged) != "new text" {
		t.Fatalf("mutating Get result changed stored content to %q", unchanged)
	}
}
