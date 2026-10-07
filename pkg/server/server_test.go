package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AobaIwaki123/lumidive/pkg/api"
	"github.com/AobaIwaki123/lumidive/pkg/ticketdive"
)

type mockTransport struct {
	responseBody string
	statusCode   int
}

func (m *mockTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(strings.NewReader(m.responseBody)),
		Header:     make(http.Header),
	}, nil
}

const mockHTML = `<!DOCTYPE html><html><head>
<script id="__NEXT_DATA__" type="application/json">
{
  "props": {
    "pageProps": {
      "__superjsonProps": {
        "json": {
          "eventDetail": {
            "event": {
              "id": "mock_id",
              "name": "Mock Live",
              "detail": "Mock detail"
            },
            "stages": [
              {
                "id": "st_1",
                "stageName": "Mock Hall",
                "startStage": "2026-10-22T06:45:00.000Z",
                "venue": { "name": "Mock Hall" }
              }
            ],
            "ticketInfoList": []
          },
          "eventImages": []
        }
      }
    }
  }
}
</script>
</head><body></body></html>`

func setupTestServer() http.Handler {
	hc := &http.Client{
		Transport: &mockTransport{
			responseBody: mockHTML,
			statusCode:   http.StatusOK,
		},
	}
	client := ticketdive.NewClient(ticketdive.WithHTTPClient(hc))
	service := ticketdive.NewCachedService(client, 1*time.Minute)
	srv := NewServer(service)

	mux := http.NewServeMux()
	return api.HandlerFromMux(srv, mux)
}

func TestHealthEndpoint(t *testing.T) {
	handler := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp api.HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("status = %q, want 'ok'", resp.Status)
	}
	if resp.Service != "lumidive" {
		t.Errorf("service = %q, want 'lumidive'", resp.Service)
	}
}

func TestGetEventById(t *testing.T) {
	handler := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/plkt1022", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp api.EventResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success true, got false")
	}
	if resp.Data.Id != "mock_id" {
		t.Errorf("expected event id 'mock_id', got %q", resp.Data.Id)
	}
}

func TestGetEventByUrl(t *testing.T) {
	handler := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events?url=https://ticketdive.com/event/plkt1022", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp api.EventResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success true")
	}
}

func TestParseEvent(t *testing.T) {
	handler := setupTestServer()

	body := []byte(`{"url": "https://ticketdive.com/event/plkt1022"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/parse", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp api.EventResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if resp.Data.Name != "Mock Live" {
		t.Errorf("expected event name 'Mock Live', got %q", resp.Data.Name)
	}
}

func TestGetEventIcal(t *testing.T) {
	handler := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/plkt1022/ical", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if !strings.Contains(rec.Body.String(), "BEGIN:VCALENDAR") {
		t.Errorf("expected iCal format, got:\n%s", rec.Body.String())
	}
}

func TestBatchParseEvents(t *testing.T) {
	handler := setupTestServer()

	body := []byte(`{"targets": ["plkt1022", "https://t-dv.com/tgg_1006"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp api.BatchEventResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success true")
	}
	if len(resp.Results) != 2 {
		t.Fatalf("expected 2 batch results, got %d", len(resp.Results))
	}
	if !resp.Results[0].Success {
		t.Errorf("expected result 0 to succeed")
	}
}

