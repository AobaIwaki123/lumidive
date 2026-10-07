// Package ticketdive provides client and parser capabilities for TicketDive events.
package ticketdive

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var eventIDRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// NormalizedURL holds the parsed components of a TicketDive event reference.
type NormalizedURL struct {
	EventID      string
	TargetURL    string
	CanonicalURL string
	ShortURL     string
}

// NormalizeInput normalizes an event ID or URL (full or shortened) into standard URLs.
func NormalizeInput(raw string) (*NormalizedURL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("empty event input")
	}

	// Direct event ID match
	if eventIDRegex.MatchString(trimmed) && !strings.Contains(trimmed, "/") && !strings.Contains(trimmed, ".") {
		return &NormalizedURL{
			EventID:      trimmed,
			TargetURL:    fmt.Sprintf("https://ticketdive.com/event/%s", trimmed),
			CanonicalURL: fmt.Sprintf("https://ticketdive.com/event/%s", trimmed),
			ShortURL:     fmt.Sprintf("https://t-dv.com/%s", trimmed),
		}, nil
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	// Handle relative or no-scheme URLs
	if u.Scheme == "" {
		u, err = url.Parse("https://" + trimmed)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}
	}

	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	var eventID string
	var targetURL string

	for i, part := range pathParts {
		if part == "event" {
			if i+1 < len(pathParts) && pathParts[i+1] == "fc" && i+2 < len(pathParts) {
				eventID = pathParts[i+2]
				targetURL = fmt.Sprintf("https://ticketdive.com/event/fc/%s", eventID)
				break
			} else if i+1 < len(pathParts) {
				eventID = pathParts[i+1]
				targetURL = fmt.Sprintf("https://ticketdive.com/event/%s", eventID)
				break
			}
		}
	}

	if eventID == "" {
		// Short URL: t-dv.com/<id>
		if u.Host == "t-dv.com" || u.Host == "www.t-dv.com" {
			if len(pathParts) > 0 && pathParts[0] != "" {
				eventID = pathParts[0]
				targetURL = fmt.Sprintf("https://ticketdive.com/event/%s", eventID)
			}
		} else if len(pathParts) > 0 && pathParts[len(pathParts)-1] != "" {
			eventID = pathParts[len(pathParts)-1]
			targetURL = u.String()
		}
	}

	if eventID == "" {
		return nil, fmt.Errorf("unable to determine event ID from %q", raw)
	}

	return &NormalizedURL{
		EventID:      eventID,
		TargetURL:    targetURL,
		CanonicalURL: fmt.Sprintf("https://ticketdive.com/event/%s", eventID),
		ShortURL:     fmt.Sprintf("https://t-dv.com/%s", eventID),
	}, nil
}
