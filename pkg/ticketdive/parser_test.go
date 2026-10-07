package ticketdive

import (
	"testing"
)

const sampleHTML = `<!DOCTYPE html><html><head>
<script id="__NEXT_DATA__" type="application/json">
{
  "props": {
    "pageProps": {
      "serverTime": 1791377301841,
      "__superjsonProps": {
        "json": {
          "eventDetail": {
            "event": {
              "id": "event_id_001",
              "name": "Sample Test Live",
              "detail": "Test live details",
              "release": "2026-10-01T00:00:00.000Z"
            },
            "stages": [
              {
                "id": "stage_001",
                "stageName": "Test Arena",
                "startStage": "2026-10-22T06:45:00.000Z",
                "openVenue": "2026-10-22T06:15:00.000Z",
                "venue": {
                  "name": "Test Arena",
                  "address": "Tokyo"
                },
                "artists": [
                  {
                    "id": "artist_001",
                    "name": "Test Idol Group"
                  }
                ]
              }
            ],
            "ticketInfoList": [
              {
                "id": "ticket_group_001",
                "name": "一般販売",
                "receptionType": "first",
                "startApply": "2026-10-02T11:00:00.000Z",
                "endApply": "2026-10-22T06:45:00.000Z",
                "paymentChannels": ["card", "konbini"],
                "status": "applied",
                "customize": [
                  {
                    "label": "お目当て",
                    "required": true,
                    "type": "select",
                    "selectOptions": [
                      {"value": "Test Idol Group"}
                    ]
                  }
                ],
                "ticketTypes": [
                  {
                    "id": "tt_001",
                    "name": "前方優先",
                    "price": 5000,
                    "remainingRate": 1.0,
                    "status": "applied",
                    "prefix": "A"
                  },
                  {
                    "id": "tt_002",
                    "name": "完売チケット",
                    "price": 2000,
                    "remainingRate": 0.0,
                    "status": "closed",
                    "prefix": "B"
                  }
                ]
              }
            ]
          },
          "eventImages": [
            {
              "imageSource": "https://storage.googleapis.com/test-bucket/flyer.jpg"
            }
          ]
        }
      }
    }
  }
}
</script>
</head><body></body></html>`

func TestParseHTML(t *testing.T) {
	norm := &NormalizedURL{
		EventID:      "test_live",
		TargetURL:    "https://ticketdive.com/event/test_live",
		CanonicalURL: "https://ticketdive.com/event/test_live",
		ShortURL:     "https://t-dv.com/test_live",
	}

	res, err := ParseHTML(sampleHTML, norm)
	if err != nil {
		t.Fatalf("ParseHTML() error = %v", err)
	}

	event := res.Event
	if event.Id != "event_id_001" {
		t.Errorf("Id = %v, want event_id_001", event.Id)
	}
	if event.Name != "Sample Test Live" {
		t.Errorf("Name = %v, want Sample Test Live", event.Name)
	}
	if event.Slug != "test_live" {
		t.Errorf("Slug = %v, want test_live", event.Slug)
	}

	// Verify stats
	if event.Stats == nil {
		t.Fatalf("expected Stats not nil")
	}
	if *event.Stats.TotalTicketTypes != 2 {
		t.Errorf("totalTicketTypes = %d, want 2", *event.Stats.TotalTicketTypes)
	}
	if *event.Stats.SoldOutTicketTypes != 1 {
		t.Errorf("soldOutTicketTypes = %d, want 1", *event.Stats.SoldOutTicketTypes)
	}
	if !*event.Stats.HasAvailableTickets {
		t.Errorf("expected hasAvailableTickets true")
	}

	// Verify source
	if res.Source == nil || *res.Source.Platform != "ticketdive" {
		t.Errorf("expected source platform ticketdive, got %v", res.Source)
	}

	// Verify stages
	if event.Stages == nil || len(*event.Stages) != 1 {
		t.Fatalf("expected 1 stage, got %v", event.Stages)
	}
	st := (*event.Stages)[0]
	if st.StageName != "Test Arena" {
		t.Errorf("stageName = %v, want Test Arena", st.StageName)
	}

	// Verify artists
	if event.Artists == nil || len(*event.Artists) != 1 {
		t.Fatalf("expected 1 artist, got %v", event.Artists)
	}
	art := (*event.Artists)[0]
	if art.Name != "Test Idol Group" {
		t.Errorf("artist name = %v, want Test Idol Group", art.Name)
	}

	// Verify ticket groups
	if event.TicketGroups == nil || len(*event.TicketGroups) != 1 {
		t.Fatalf("expected 1 ticket group, got %v", event.TicketGroups)
	}
	tg := (*event.TicketGroups)[0]
	if tg.Name != "一般販売" {
		t.Errorf("ticket group name = %v, want 一般販売", tg.Name)
	}
	if len(tg.Types) != 2 {
		t.Fatalf("expected 2 ticket types, got %d", len(tg.Types))
	}

	// First type: available
	if tg.Types[0].IsSoldOut {
		t.Errorf("expected first ticket not sold out, got sold out")
	}
	if tg.Types[0].Price != 5000 {
		t.Errorf("price = %d, want 5000", tg.Types[0].Price)
	}

	// Second type: sold out
	if !tg.Types[1].IsSoldOut {
		t.Errorf("expected second ticket sold out, got not sold out")
	}

	// Verify images
	if event.Images == nil || len(*event.Images) != 1 {
		t.Fatalf("expected 1 image, got %v", event.Images)
	}
	if (*event.Images)[0] != "https://storage.googleapis.com/test-bucket/flyer.jpg" {
		t.Errorf("image URL mismatch: %v", (*event.Images)[0])
	}
}

func TestGenerateICal(t *testing.T) {
	norm := &NormalizedURL{
		EventID:      "test_live",
		CanonicalURL: "https://ticketdive.com/event/test_live",
	}
	res, err := ParseHTML(sampleHTML, norm)
	if err != nil {
		t.Fatalf("ParseHTML() failed: %v", err)
	}

	cal, err := GenerateICal(res.Event)
	if err != nil {
		t.Fatalf("GenerateICal() error = %v", err)
	}

	if cal == "" {
		t.Fatal("GenerateICal() returned empty string")
	}

	expectedSubstrings := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"BEGIN:VEVENT",
		"SUMMARY:Sample Test Live",
		"LOCATION:Test Arena",
		"END:VEVENT",
		"END:VCALENDAR",
	}

	for _, sub := range expectedSubstrings {
		if !contains(cal, sub) {
			t.Errorf("expected ical to contain %q, but got:\n%s", sub, cal)
		}
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || len(needle) == 0 || (len(haystack) > 0 && len(needle) > 0 && stringContains(haystack, needle)))
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
