package session

import (
	"sync"
)

type SessionStore struct {
	mu   sync.RWMutex
	Data map[int64]map[string]interface{}
}

var GlobalStore *SessionStore

func init() {
	GlobalStore = NewSessionStore()
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		Data: make(map[int64]map[string]interface{}),
	}
}

func (s *SessionStore) Get(userID int64, key string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if userStore, ok := s.Data[userID]; ok {
		return userStore[key]
	}
	return nil
}

func (s *SessionStore) Set(userID int64, key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.Data[userID]; !ok {
		s.Data[userID] = make(map[string]interface{})
	}
	s.Data[userID][key] = value
}

func (s *SessionStore) Delete(userID int64, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if userStore, ok := s.Data[userID]; ok {
		delete(userStore, key)
	}
}
