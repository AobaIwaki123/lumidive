// Package server provides HTTP handler and API server for lumidive.
package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/AobaIwaki123/lumidive/pkg/api"
	"github.com/AobaIwaki123/lumidive/pkg/ticketdive"
)

// Server implements api.ServerInterface.
type Server struct {
	service *ticketdive.CachedService
}

// NewServer creates a new API server.
func NewServer(service *ticketdive.CachedService) *Server {
	return &Server{
		service: service,
	}
}

// GetHealth returns health status.
// (GET /healthz)
func (s *Server) GetHealth(w http.ResponseWriter, _ *http.Request) {
	resp := api.HealthResponse{
		Status:    "ok",
		Service:   "lumidive",
		Timestamp: time.Now().UnixMilli(),
	}
	s.writeJSON(w, http.StatusOK, resp)
}

// GetEventById fetches event metadata by event ID.
// (GET /api/v1/events/{eventId})
func (s *Server) GetEventById(w http.ResponseWriter, r *http.Request, eventID string) {
	if eventID == "" {
		s.writeError(w, http.StatusBadRequest, "event ID is required")
		return
	}

	event, cached, err := s.service.GetEvent(r.Context(), eventID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, api.EventResponse{
		Success: true,
		Cached:  cached,
		Data:    *event,
	})
}

// GetEventByUrl fetches event metadata by URL query parameter.
// (GET /api/v1/events)
func (s *Server) GetEventByUrl(w http.ResponseWriter, r *http.Request, params api.GetEventByUrlParams) {
	if params.Url == "" {
		s.writeError(w, http.StatusBadRequest, "query parameter 'url' is required")
		return
	}

	event, cached, err := s.service.GetEvent(r.Context(), params.Url)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, api.EventResponse{
		Success: true,
		Cached:  cached,
		Data:    *event,
	})
}

// ParseEvent parses an event from POST request body.
// (POST /api/v1/events/parse)
func (s *Server) ParseEvent(w http.ResponseWriter, r *http.Request) {
	var req api.ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	target := ""
	if req.Url != nil && *req.Url != "" {
		target = *req.Url
	} else if req.Id != nil && *req.Id != "" {
		target = *req.Id
	}

	if target == "" {
		s.writeError(w, http.StatusBadRequest, "either 'url' or 'id' must be provided")
		return
	}

	event, cached, err := s.service.GetEvent(r.Context(), target)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, api.EventResponse{
		Success: true,
		Cached:  cached,
		Data:    *event,
	})
}

// GetEventIcal returns iCalendar format for the event.
// (GET /api/v1/events/{eventId}.ics)
func (s *Server) GetEventIcal(w http.ResponseWriter, r *http.Request, eventID string) {
	if eventID == "" {
		s.writeError(w, http.StatusBadRequest, "event ID is required")
		return
	}

	event, _, err := s.service.GetEvent(r.Context(), eventID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	calData, err := ticketdive.GenerateICal(event)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(calData))
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, api.ErrorResponse{
		Success: false,
		Error:   message,
	})
}
