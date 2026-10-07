// Package server provides HTTP handler and API server for lumidive.
package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/AobaIwaki123/lumidive/pkg/api"
	"github.com/AobaIwaki123/lumidive/pkg/ticketdive"
)

const currentAPIVersion = "1.0"

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
		Version:   "1.0.0",
		Timestamp: time.Now().UnixMilli(),
	}
	s.writeJSON(w, http.StatusOK, resp)
}

// GetArtistById fetches artist profile and events by artist slug or ID.
// (GET /api/v1/artists/{artistId})
func (s *Server) GetArtistById(w http.ResponseWriter, r *http.Request, artistID string, params api.GetArtistByIdParams) {
	if artistID == "" {
		s.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "artist ID is required", nil)
		return
	}

	refresh := false
	if params.Refresh != nil {
		refresh = *params.Refresh
	}

	artist, cached, source, cachedAt, err := s.service.GetArtist(r.Context(), artistID, refresh)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "UPSTREAM_ERROR", err.Error(), nil)
		return
	}

	s.writeJSON(w, http.StatusOK, api.ArtistDetailResponse{
		ApiVersion: currentAPIVersion,
		Success:    true,
		Cached:     cached,
		CachedAt:   cachedAt,
		Source:     source,
		Data:       *artist,
	})
}

// GetArtistIcal returns an aggregated iCalendar feed of all events for the artist.
// (GET /api/v1/artists/{artistId}/ical)
func (s *Server) GetArtistIcal(w http.ResponseWriter, r *http.Request, artistID string) {
	if artistID == "" {
		s.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "artist ID is required", nil)
		return
	}

	artist, _, _, _, err := s.service.GetArtist(r.Context(), artistID, false)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "UPSTREAM_ERROR", err.Error(), nil)
		return
	}

	calData, err := ticketdive.GenerateArtistICal(artist)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "ICAL_ERROR", err.Error(), nil)
		return
	}

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(calData))
}

// GetEventById fetches event metadata by event ID.
// (GET /api/v1/events/{eventId})
func (s *Server) GetEventById(w http.ResponseWriter, r *http.Request, eventID string, params api.GetEventByIdParams) {
	if eventID == "" {
		s.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "event ID is required", nil)
		return
	}

	refresh := false
	if params.Refresh != nil {
		refresh = *params.Refresh
	}

	event, cached, source, cachedAt, err := s.service.GetEvent(r.Context(), eventID, refresh)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "UPSTREAM_ERROR", err.Error(), nil)
		return
	}

	s.writeJSON(w, http.StatusOK, api.EventResponse{
		ApiVersion: currentAPIVersion,
		Success:    true,
		Cached:     cached,
		CachedAt:   cachedAt,
		Source:     source,
		Data:       *event,
	})
}

// GetEventByUrl fetches event metadata by URL query parameter.
// (GET /api/v1/events)
func (s *Server) GetEventByUrl(w http.ResponseWriter, r *http.Request, params api.GetEventByUrlParams) {
	if params.Url == "" {
		s.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "query parameter 'url' is required", nil)
		return
	}

	refresh := false
	if params.Refresh != nil {
		refresh = *params.Refresh
	}

	event, cached, source, cachedAt, err := s.service.GetEvent(r.Context(), params.Url, refresh)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "UPSTREAM_ERROR", err.Error(), nil)
		return
	}

	s.writeJSON(w, http.StatusOK, api.EventResponse{
		ApiVersion: currentAPIVersion,
		Success:    true,
		Cached:     cached,
		CachedAt:   cachedAt,
		Source:     source,
		Data:       *event,
	})
}

// ParseEvent parses an event from POST request body.
// (POST /api/v1/events/parse)
func (s *Server) ParseEvent(w http.ResponseWriter, r *http.Request) {
	var req api.ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid JSON payload", nil)
		return
	}

	target := ""
	if req.Url != nil && *req.Url != "" {
		target = *req.Url
	} else if req.Id != nil && *req.Id != "" {
		target = *req.Id
	}

	if target == "" {
		s.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "either 'url' or 'id' must be provided", nil)
		return
	}

	refresh := false
	if req.Refresh != nil {
		refresh = *req.Refresh
	}

	event, cached, source, cachedAt, err := s.service.GetEvent(r.Context(), target, refresh)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "UPSTREAM_ERROR", err.Error(), nil)
		return
	}

	s.writeJSON(w, http.StatusOK, api.EventResponse{
		ApiVersion: currentAPIVersion,
		Success:    true,
		Cached:     cached,
		CachedAt:   cachedAt,
		Source:     source,
		Data:       *event,
	})
}

// BatchParseEvents parses multiple events in a single batch.
// (POST /api/v1/events/batch)
func (s *Server) BatchParseEvents(w http.ResponseWriter, r *http.Request) {
	var req api.BatchParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid JSON payload", nil)
		return
	}

	if len(req.Targets) == 0 {
		s.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "targets array must not be empty", nil)
		return
	}

	refresh := false
	if req.Refresh != nil {
		refresh = *req.Refresh
	}

	results := make([]api.BatchEventResult, len(req.Targets))
	var wg sync.WaitGroup

	for i, target := range req.Targets {
		wg.Add(1)
		go func(idx int, tgt string) {
			defer wg.Done()
			ev, _, _, _, err := s.service.GetEvent(r.Context(), tgt, refresh)
			if err != nil {
				errMsg := err.Error()
				results[idx] = api.BatchEventResult{
					Target:  tgt,
					Success: false,
					Error:   &errMsg,
				}
				return
			}
			results[idx] = api.BatchEventResult{
				Target:  tgt,
				Success: true,
				Event:   ev,
			}
		}(i, target)
	}

	wg.Wait()

	s.writeJSON(w, http.StatusOK, api.BatchEventResponse{
		ApiVersion: currentAPIVersion,
		Success:    true,
		Results:    results,
	})
}

// GetEventIcal returns iCalendar format for the event.
// (GET /api/v1/events/{eventId}/ical)
func (s *Server) GetEventIcal(w http.ResponseWriter, r *http.Request, eventID string) {
	if eventID == "" {
		s.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "event ID is required", nil)
		return
	}

	event, _, _, _, err := s.service.GetEvent(r.Context(), eventID, false)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "UPSTREAM_ERROR", err.Error(), nil)
		return
	}

	calData, err := ticketdive.GenerateICal(event)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "ICAL_ERROR", err.Error(), nil)
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

func (s *Server) writeError(w http.ResponseWriter, status int, code, message string, details []string) {
	resp := api.ErrorResponse{
		ApiVersion: currentAPIVersion,
		Success:    false,
		Code:       &code,
		Error:      message,
	}
	if len(details) > 0 {
		resp.Details = &details
	}
	s.writeJSON(w, status, resp)
}
