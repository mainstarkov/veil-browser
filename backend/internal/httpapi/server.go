package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mainstarkov/veil-browser/backend/internal/session"
)

const (
	sessionCookieName = "veil_session"
	maxRequestBody    = 1 << 20
)

type Options struct {
	AllowedOrigin string
	SecureCookies bool
}

type API struct {
	store  *session.Store
	option Options
	mux    *http.ServeMux
}

type responseEnvelope struct {
	Data any `json:"data,omitempty"`
	Meta any `json:"meta,omitempty"`
}

type errorEnvelope struct {
	Error apiError `json:"error"`
	Meta  any      `json:"meta,omitempty"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func New(store *session.Store, option Options) http.Handler {
	api := &API{store: store, option: option, mux: http.NewServeMux()}
	api.routes()
	return api.middleware(api.mux)
}

func (api *API) routes() {
	api.mux.HandleFunc("GET /health", api.health)
	api.mux.HandleFunc("POST /api/v1/sessions", api.createSession)
	api.mux.HandleFunc("GET /api/v1/sessions/current", api.getSession)
	api.mux.HandleFunc("PUT /api/v1/sessions/current", api.updateSession)
	api.mux.HandleFunc("DELETE /api/v1/sessions/current", api.deleteSession)
}

func (api *API) health(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, responseEnvelope{Data: map[string]string{"status": "ok"}})
}

func (api *API) createSession(writer http.ResponseWriter, request *http.Request) {
	if cookie, err := request.Cookie(sessionCookieName); err == nil {
		if _, err := api.store.Get(cookie.Value); err == nil {
			writeError(writer, request, http.StatusConflict, "SESSION_EXISTS", "session already exists")
			return
		}
		api.clearSessionCookie(writer)
	}

	created, err := api.store.Create()
	if err != nil {
		writeError(writer, request, http.StatusInternalServerError, "INTERNAL_ERROR", "could not create session")
		return
	}
	api.setSessionCookie(writer, created.ID, time.Until(created.ExpiresAt))
	writeJSON(writer, http.StatusCreated, responseEnvelope{Data: created, Meta: requestMeta(request)})
}

func (api *API) getSession(writer http.ResponseWriter, request *http.Request) {
	current, ok := api.currentSession(writer, request)
	if !ok {
		return
	}
	writeJSON(writer, http.StatusOK, responseEnvelope{Data: current, Meta: requestMeta(request)})
}

func (api *API) updateSession(writer http.ResponseWriter, request *http.Request) {
	current, ok := api.currentSession(writer, request)
	if !ok {
		return
	}

	var state session.State
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, maxRequestBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&state); err != nil {
		writeError(writer, request, http.StatusBadRequest, "INVALID_JSON", "request body is invalid")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(writer, request, http.StatusBadRequest, "INVALID_JSON", "request body must contain one JSON object")
		return
	}

	updated, err := api.store.Update(current.ID, state)
	if err != nil {
		if errors.Is(err, session.ErrNotFound) {
			api.clearSessionCookie(writer)
			writeError(writer, request, http.StatusNotFound, "SESSION_NOT_FOUND", "session does not exist or has expired")
			return
		}
		writeError(writer, request, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	api.setSessionCookie(writer, updated.ID, time.Until(updated.ExpiresAt))
	writeJSON(writer, http.StatusOK, responseEnvelope{Data: updated, Meta: requestMeta(request)})
}

func (api *API) deleteSession(writer http.ResponseWriter, request *http.Request) {
	cookie, err := request.Cookie(sessionCookieName)
	if err == nil {
		api.store.Delete(cookie.Value)
	}
	api.clearSessionCookie(writer)
	writer.WriteHeader(http.StatusNoContent)
}

func (api *API) currentSession(writer http.ResponseWriter, request *http.Request) (session.Session, bool) {
	cookie, err := request.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		writeError(writer, request, http.StatusNotFound, "SESSION_NOT_FOUND", "session does not exist or has expired")
		return session.Session{}, false
	}
	current, err := api.store.Get(cookie.Value)
	if err != nil {
		api.clearSessionCookie(writer)
		writeError(writer, request, http.StatusNotFound, "SESSION_NOT_FOUND", "session does not exist or has expired")
		return session.Session{}, false
	}
	return current, true
}

func (api *API) setSessionCookie(writer http.ResponseWriter, id string, ttl time.Duration) {
	http.SetCookie(writer, &http.Cookie{
		Name:     sessionCookieName,
		Value:    id,
		Path:     "/api/v1/sessions",
		MaxAge:   max(1, int(ttl.Seconds())),
		HttpOnly: true,
		Secure:   api.option.SecureCookies,
		SameSite: http.SameSiteStrictMode,
	})
}

func (api *API) clearSessionCookie(writer http.ResponseWriter) {
	http.SetCookie(writer, &http.Cookie{
		Name:     sessionCookieName,
		Path:     "/api/v1/sessions",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   api.option.SecureCookies,
		SameSite: http.SameSiteStrictMode,
	})
}

func (api *API) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestID := request.Header.Get("X-Request-ID")
		if requestID == "" || len(requestID) > 128 {
			requestID = newRequestID()
		}
		request.Header.Set("X-Request-ID", requestID)
		writer.Header().Set("X-Request-ID", requestID)
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("Cache-Control", "no-store")
		writer.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		origin := request.Header.Get("Origin")
		if origin != "" {
			if origin != api.option.AllowedOrigin {
				writeError(writer, request, http.StatusForbidden, "ORIGIN_NOT_ALLOWED", "request origin is not allowed")
				return
			}
			writer.Header().Set("Access-Control-Allow-Origin", origin)
			writer.Header().Set("Access-Control-Allow-Credentials", "true")
			writer.Header().Set("Vary", "Origin")
		}

		if request.Method == http.MethodOptions {
			writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
			writer.Header().Set("Access-Control-Max-Age", "600")
			writer.WriteHeader(http.StatusNoContent)
			return
		}

		if (request.Method == http.MethodPost || request.Method == http.MethodPut) &&
			!strings.HasPrefix(request.Header.Get("Content-Type"), "application/json") && request.ContentLength != 0 {
			writeError(writer, request, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Content-Type must be application/json")
			return
		}

		started := time.Now()
		next.ServeHTTP(writer, request)
		slog.Info("request", "id", requestID, "method", request.Method, "path", request.URL.Path, "duration", time.Since(started))
	})
}

func requestMeta(request *http.Request) map[string]string {
	return map[string]string{"requestId": request.Header.Get("X-Request-ID")}
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		slog.Error("could not encode response", "error", err)
	}
}

func writeError(writer http.ResponseWriter, request *http.Request, status int, code, message string) {
	writeJSON(writer, status, errorEnvelope{
		Error: apiError{Code: code, Message: message},
		Meta:  requestMeta(request),
	})
}

func newRequestID() string {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "request-unavailable"
	}
	return hex.EncodeToString(buffer)
}
