package internal

import (
	"errors"
	"sync"

	"github.com/tsopia/go-kit/storage/providers"
)

// MultipartState 保存 multipart uploadID 到对象 key 的映射。
type MultipartState struct {
	mu   sync.RWMutex
	keys map[string]string
}

func NewMultipartState() *MultipartState {
	return &MultipartState{
		keys: make(map[string]string),
	}
}

func (s *MultipartState) Store(uploadID, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[uploadID] = key
}

func (s *MultipartState) Load(uploadID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key, ok := s.keys[uploadID]
	return key, ok
}

func (s *MultipartState) Delete(uploadID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.keys, uploadID)
}

func ExistsFromError(err error) (bool, error) {
	if err == nil {
		return true, nil
	}
	if errors.Is(err, providers.ErrObjectNotFound) {
		return false, nil
	}
	return false, err
}
