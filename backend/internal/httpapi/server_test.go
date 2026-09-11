package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mainstarkov/veil-browser/backend/internal/session"
)

func TestSessionAPI(t *testing.T) {
	server := httptest.NewServer(New(session.NewStore(30*time.Minute), Options{
		AllowedOrigin: "http://localhost:3000",
	}))
	defer server.Close()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New() error = %v", err)
	}
	client := server.Client()
	client.Jar = jar

	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/sessions", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	request.Header.Set("Origin", "http://localhost:3000")
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("create request error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", response.StatusCode, http.StatusCreated)
	}
	if response.Header.Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatal("expected allowed CORS origin")
	}
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	var createdEnvelope struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &createdEnvelope); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if _, exposed := createdEnvelope.Data["id"]; exposed {
		t.Fatal("response exposed the private session id")
	}
	cookies := response.Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("session cookie is not hardened: %+v", cookies)
	}

	response, err = client.Get(server.URL + "/api/v1/sessions/current")
	if err != nil {
		t.Fatalf("get request error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	request, err = http.NewRequest(http.MethodDelete, server.URL+"/api/v1/sessions/current", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	response, err = client.Do(request)
	if err != nil {
		t.Fatalf("delete request error = %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
}

func TestUpdateRejectsUnsafeState(t *testing.T) {
	store := session.NewStore(30 * time.Minute)
	created, err := store.Create()
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	created.State.Tabs[0].URL = "file:///etc/passwd"
	body, err := json.Marshal(created.State)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodPut, "/api/v1/sessions/current", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: created.ID})
	recorder := httptest.NewRecorder()
	New(store, Options{AllowedOrigin: "http://localhost:3000"}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestRejectsUnknownOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set("Origin", "https://attacker.example")
	recorder := httptest.NewRecorder()
	New(session.NewStore(time.Minute), Options{AllowedOrigin: "http://localhost:3000"}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}
