package session

import (
	"errors"
	"testing"
	"time"
)

func TestStoreLifecycle(t *testing.T) {
	store := NewStore(30 * time.Minute)
	created, err := store.Create()
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := created.State.Validate(); err != nil {
		t.Fatalf("default state is invalid: %v", err)
	}

	created.State.Tabs[0].Title = "Example"
	updated, err := store.Update(created.ID, created.State)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.State.Tabs[0].Title != "Example" {
		t.Fatalf("Update() title = %q", updated.State.Tabs[0].Title)
	}

	store.Delete(created.ID)
	if _, err := store.Get(created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want ErrNotFound", err)
	}
}

func TestStoreExpiresSessions(t *testing.T) {
	store := NewStore(time.Minute)
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	created, err := store.Create()
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	now = now.Add(2 * time.Minute)
	if _, err := store.Get(created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() expired error = %v, want ErrNotFound", err)
	}
}

func TestStateValidationRejectsUnsafeURL(t *testing.T) {
	state := DefaultState("tab-1")
	state.Tabs[0].URL = "javascript:alert(1)"
	if err := state.Validate(); err == nil {
		t.Fatal("Validate() accepted an unsafe URL")
	}
}

func TestStoreReturnsIndependentState(t *testing.T) {
	store := NewStore(time.Minute)
	created, err := store.Create()
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	created.State.Tabs[0].Title = "changed outside store"

	stored, err := store.Get(created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored.State.Tabs[0].Title == created.State.Tabs[0].Title {
		t.Fatal("store exposed mutable internal state")
	}
}
