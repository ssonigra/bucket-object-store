// Package store keeps objects in memory and shares identical content within
// each bucket.
package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
)

var ErrHashCollision = errors.New("different object contents produced the same SHA-256 hash")

// Store maps each bucket and object ID to content stored once inside that
// bucket. A separate blob map per bucket ensures that buckets never share
// stored data with one another.
type Store struct {
	mu      sync.RWMutex
	buckets map[string]*bucket
}

type bucket struct {
	objects map[string]string // object ID -> content hash
	blobs   map[string][]byte // content hash -> content
	refs    map[string]int    // content hash -> number of object IDs using it
}

func New() *Store {
	return &Store{buckets: make(map[string]*bucket)}
}

// Put stores content under an object ID. If another object in the same bucket
// already has identical bytes, both IDs point to the same in-memory copy.
func (s *Store) Put(bucketName, objectID string, data []byte) error {
	hashBytes := sha256.Sum256(data)
	hash := hex.EncodeToString(hashBytes[:])

	s.mu.Lock()
	defer s.mu.Unlock()

	b, ok := s.buckets[bucketName]
	if !ok {
		b = &bucket{
			objects: make(map[string]string),
			blobs:   make(map[string][]byte),
			refs:    make(map[string]int),
		}
		s.buckets[bucketName] = b
	}

	if previousHash, exists := b.objects[objectID]; exists && previousHash == hash {
		return nil
	}

	if existing, exists := b.blobs[hash]; exists {
		// SHA-256 collisions are extraordinarily unlikely. Checking the bytes
		// makes sure a collision could never cause one object's data to be
		// returned in place of another object's data.
		if !bytes.Equal(existing, data) {
			return ErrHashCollision
		}
	} else {
		b.blobs[hash] = append([]byte(nil), data...)
	}

	if previousHash, exists := b.objects[objectID]; exists {
		s.releaseBlob(b, previousHash)
	}
	b.objects[objectID] = hash
	b.refs[hash]++
	return nil
}

// Get returns a copy of an object's content so callers cannot modify stored
// data after the store lock has been released.
func (s *Store) Get(bucketName, objectID string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	b, exists := s.buckets[bucketName]
	if !exists {
		return nil, false
	}
	hash, exists := b.objects[objectID]
	if !exists {
		return nil, false
	}
	return append([]byte(nil), b.blobs[hash]...), true
}

// Delete removes an object ID and frees its stored content when that was the
// last ID in the bucket pointing to the content.
func (s *Store) Delete(bucketName, objectID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, exists := s.buckets[bucketName]
	if !exists {
		return false
	}
	hash, exists := b.objects[objectID]
	if !exists {
		return false
	}

	delete(b.objects, objectID)
	s.releaseBlob(b, hash)
	if len(b.objects) == 0 {
		delete(s.buckets, bucketName)
	}
	return true
}

func (s *Store) releaseBlob(b *bucket, hash string) {
	b.refs[hash]--
	if b.refs[hash] == 0 {
		delete(b.refs, hash)
		delete(b.blobs, hash)
	}
}
