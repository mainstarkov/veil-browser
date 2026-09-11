package session

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

var ErrNotFound = errors.New("session not found")

type Store struct {
	mu       sync.RWMutex
	sessions map[string]Session
	ttl      time.Duration
	now      func() time.Time
}

func NewStore(ttl time.Duration) *Store {
	return &Store{
		sessions: make(map[string]Session),
		ttl:      ttl,
		now:      time.Now,
	}
}

func (store *Store) Create() (Session, error) {
	id, err := randomID()
	if err != nil {
		return Session{}, err
	}
	tabID, err := randomID()
	if err != nil {
		return Session{}, err
	}
	now := store.now().UTC()
	created := Session{
		ID:        id,
		State:     DefaultState(tabID),
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(store.ttl),
	}

	store.mu.Lock()
	store.sessions[id] = cloneSession(created)
	store.mu.Unlock()
	return cloneSession(created), nil
}

func (store *Store) Get(id string) (Session, error) {
	store.mu.RLock()
	current, ok := store.sessions[id]
	store.mu.RUnlock()
	if !ok {
		return Session{}, ErrNotFound
	}
	if !current.ExpiresAt.After(store.now()) {
		store.Delete(id)
		return Session{}, ErrNotFound
	}
	return cloneSession(current), nil
}

func (store *Store) Update(id string, state State) (Session, error) {
	if err := state.Validate(); err != nil {
		return Session{}, err
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	current, ok := store.sessions[id]
	if !ok || !current.ExpiresAt.After(store.now()) {
		delete(store.sessions, id)
		return Session{}, ErrNotFound
	}
	now := store.now().UTC()
	current.State = cloneState(state)
	current.UpdatedAt = now
	current.ExpiresAt = now.Add(store.ttl)
	store.sessions[id] = current
	return cloneSession(current), nil
}

func (store *Store) Delete(id string) {
	store.mu.Lock()
	delete(store.sessions, id)
	store.mu.Unlock()
}

func (store *Store) DeleteExpired() int {
	now := store.now()
	deleted := 0
	store.mu.Lock()
	for id, current := range store.sessions {
		if !current.ExpiresAt.After(now) {
			delete(store.sessions, id)
			deleted++
		}
	}
	store.mu.Unlock()
	return deleted
}

func randomID() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func cloneSession(current Session) Session {
	current.State = cloneState(current.State)
	return current
}

func cloneState(state State) State {
	copyState := state
	copyState.Tabs = make([]Tab, len(state.Tabs))
	for index, tab := range state.Tabs {
		copyState.Tabs[index] = tab
		copyState.Tabs[index].History = append([]string(nil), tab.History...)
	}
	return copyState
}
