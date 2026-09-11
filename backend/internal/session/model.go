package session

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	maxTabs         = 50
	maxHistoryItems = 200
	maxTitleLength  = 200
	maxURLLength    = 4096
)

type PrivacySettings struct {
	Trackers    bool `json:"trackers"`
	Cookies     bool `json:"cookies"`
	Fingerprint bool `json:"fingerprint"`
	WebRTC      bool `json:"webrtc"`
}

type Tab struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	URL          string   `json:"url"`
	History      []string `json:"history"`
	HistoryIndex int      `json:"historyIndex"`
	Private      bool     `json:"private"`
}

type State struct {
	Tabs        []Tab           `json:"tabs"`
	ActiveTabID string          `json:"activeTabId"`
	Settings    PrivacySettings `json:"settings"`
}

type Session struct {
	ID        string    `json:"-"`
	State     State     `json:"state"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func DefaultState(tabID string) State {
	return State{
		Tabs: []Tab{{
			ID:           tabID,
			Title:        "Новая вкладка",
			URL:          "veil://start",
			History:      []string{"veil://start"},
			HistoryIndex: 0,
		}},
		ActiveTabID: tabID,
		Settings: PrivacySettings{
			Trackers:    true,
			Cookies:     true,
			Fingerprint: true,
			WebRTC:      true,
		},
	}
}

func (state State) Validate() error {
	if len(state.Tabs) == 0 || len(state.Tabs) > maxTabs {
		return fmt.Errorf("tabs must contain between 1 and %d items", maxTabs)
	}

	seen := make(map[string]struct{}, len(state.Tabs))
	activeTabFound := false
	for _, tab := range state.Tabs {
		if tab.ID == "" || len(tab.ID) > 128 {
			return errors.New("tab id is invalid")
		}
		if _, exists := seen[tab.ID]; exists {
			return errors.New("tab ids must be unique")
		}
		seen[tab.ID] = struct{}{}
		activeTabFound = activeTabFound || tab.ID == state.ActiveTabID

		if len(tab.Title) > maxTitleLength {
			return errors.New("tab title is too long")
		}
		if len(tab.History) == 0 || len(tab.History) > maxHistoryItems {
			return fmt.Errorf("tab history must contain between 1 and %d items", maxHistoryItems)
		}
		if tab.HistoryIndex < 0 || tab.HistoryIndex >= len(tab.History) {
			return errors.New("history index is out of range")
		}
		if err := validateBrowserURL(tab.URL); err != nil {
			return fmt.Errorf("tab url: %w", err)
		}
		for _, item := range tab.History {
			if err := validateBrowserURL(item); err != nil {
				return fmt.Errorf("history url: %w", err)
			}
		}
	}

	if !activeTabFound {
		return errors.New("active tab does not exist")
	}
	return nil
}

func validateBrowserURL(value string) error {
	if value == "veil://start" {
		return nil
	}
	if value == "" || len(value) > maxURLLength || strings.ContainsAny(value, "\r\n") {
		return errors.New("url is invalid")
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return errors.New("only http and https urls are allowed")
	}
	return nil
}
