package api

import (
	"encoding/json"
	"net/http"

	"performance-engine/internal/engine"
)

type Handler struct {
	Sender engine.Sender
	APIKey string
}

func (h Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path == "/healthz" && request.Method == http.MethodGet {
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	if h.APIKey != "" && request.Header.Get("X-API-Key") != h.APIKey {
		writeJSON(writer, http.StatusUnauthorized, map[string]string{"error": "invalid API key"})
		return
	}
	if request.Method != http.MethodPost {
		writeJSON(writer, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	switch request.URL.Path {
	case "/api/v1/send":
		h.send(writer, request)
	case "/api/v1/receive":
		receive(writer, request)
	default:
		writeJSON(writer, http.StatusNotFound, map[string]string{"error": "route not found"})
	}
}

func (h Handler) send(writer http.ResponseWriter, request *http.Request) {
	var payload engine.Request
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid request JSON"})
		return
	}
	if h.Sender == nil {
		writeJSON(writer, http.StatusServiceUnavailable, map[string]string{"error": "sender is not configured"})
		return
	}
	response, err := h.Sender.Send(request.Context(), payload)
	if err != nil {
		writeJSON(writer, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(writer, http.StatusOK, response)
}

func receive(writer http.ResponseWriter, request *http.Request) {
	var payload engine.Request
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid request JSON"})
		return
	}
	if payload.SessionID == "" {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "sessionId is required"})
		return
	}
	writeJSON(writer, http.StatusOK, engine.Response{SessionID: payload.SessionID, Response: "END OK", Continue: false})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
