package ticketdive

import (
	"strings"
	"time"

	"github.com/AobaIwaki123/lumi-spec/event"
	"github.com/AobaIwaki123/lumidive/pkg/api"
)

// ToEventCandidate maps the source-native TicketDive event to lumi-spec.
// TicketDive URLs are preserved as explicit ticket links in addition to the
// canonical source reference so downstream consumers can render both.
func ToEventCandidate(item *api.Event) event.Candidate {
	if item == nil {
		return event.Candidate{}
	}
	startAt, endAt, venue, performers := eventDetails(item)
	recordID := strings.TrimSpace(item.Id)
	if recordID == "" {
		recordID = strings.TrimSpace(item.Slug)
	}
	if recordID == "" {
		recordID = "event"
	}
	links := ticketLinks(item.Url, item.ShortUrl)
	sourceURL := strings.TrimSpace(item.Url)
	if sourceURL == "" {
		sourceURL = strings.TrimSpace(item.ShortUrl)
	}
	return event.Candidate{
		SchemaVersion: event.CandidateSchemaVersion,
		ID:            "lumidive:" + recordID,
		SourceRef: event.SourceRef{
			Source:   "lumidive",
			RecordID: recordID,
			URL:      sourceURL,
		},
		Title:      item.Name,
		LocalDate:  localDate(startAt),
		StartsAt:   startAt,
		EndsAt:     endAt,
		Venue:      venue,
		Performers: performers,
		Links:      links,
	}
}

// ToEventFeed wraps one TicketDive event in the shared feed envelope used by
// the CLI and by callers that process multiple source adapters uniformly.
func ToEventFeed(item *api.Event) event.Feed {
	feed := event.Feed{
		SchemaVersion: event.FeedSchemaVersion,
		Source:        "lumidive",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Events:        make([]event.Candidate, 0, 1),
	}
	candidate := ToEventCandidate(item)
	if candidate.ID != "" {
		feed.Events = append(feed.Events, candidate)
	}
	return feed
}

// ToArtistEventFeed converts the upcoming events exposed on a TicketDive
// artist profile to the same shared event-feed contract.
func ToArtistEventFeed(artist *api.ArtistDetail) event.Feed {
	feed := event.Feed{
		SchemaVersion: event.FeedSchemaVersion,
		Source:        "lumidive",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Events:        make([]event.Candidate, 0),
	}
	if artist == nil || artist.Events == nil {
		return feed
	}
	feed.Events = make([]event.Candidate, 0, len(*artist.Events))
	for _, item := range *artist.Events {
		recordID := strings.TrimSpace(item.Id)
		if recordID == "" {
			recordID = stringValue(item.Slug)
		}
		if recordID == "" {
			continue
		}
		startAt := stringValue(item.StartAt)
		endAt := stringValue(item.EndAt)
		feed.Events = append(feed.Events, event.Candidate{
			SchemaVersion: event.CandidateSchemaVersion,
			ID:            "lumidive:" + recordID,
			SourceRef: event.SourceRef{
				Source:   "lumidive",
				RecordID: recordID,
				URL:      stringValue(item.Url),
			},
			Title:      item.Title,
			LocalDate:  localDate(startAt),
			StartsAt:   startAt,
			EndsAt:     endAt,
			Venue:      stringValue(item.VenueName),
			Performers: []string{artist.Name},
			Links:      ticketLinks(stringValue(item.Url), ""),
		})
	}
	return feed
}

func eventDetails(item *api.Event) (startAt, endAt, venue string, performers []string) {
	if item.Stages != nil && len(*item.Stages) > 0 {
		stage := (*item.Stages)[0]
		startAt = stringValue(stage.StartAt)
		endAt = stringValue(stage.EndAt)
		if stage.Venue != nil {
			venue = stringValue(stage.Venue.Name)
		}
	}
	if item.Artists != nil {
		performers = make([]string, 0, len(*item.Artists))
		for _, artist := range *item.Artists {
			if name := strings.TrimSpace(artist.Name); name != "" {
				performers = append(performers, name)
			}
		}
	}
	return startAt, endAt, venue, performers
}

func ticketLinks(canonicalURL, shortURL string) []event.Link {
	links := make([]event.Link, 0, 2)
	seen := make(map[string]struct{})
	for _, item := range []struct {
		url    string
		label  string
		origin string
	}{
		{canonicalURL, "TicketDive", "structured_url"},
		{shortURL, "TicketDive short URL", "structured_url"},
	} {
		item.url = strings.TrimSpace(item.url)
		if item.url == "" {
			continue
		}
		if _, ok := seen[item.url]; ok {
			continue
		}
		seen[item.url] = struct{}{}
		links = append(links, event.Link{Kind: event.LinkKindTicket, URL: item.url, Label: item.label, Origin: item.origin})
	}
	return links
}

func localDate(value string) string {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, value)
	}
	if err != nil {
		return ""
	}
	return parsed.Format("2006-01-02")
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
