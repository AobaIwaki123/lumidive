// Package ticketdive provides client and parser capabilities for TicketDive events.
package ticketdive

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var idRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// TargetKind represents whether the target is an event or an artist.
type TargetKind string

const (
	KindEvent  TargetKind = "event"
	KindArtist TargetKind = "artist"
)

// NormalizedURL holds the parsed components of a TicketDive event or artist reference.
type NormalizedURL struct {
	Kind         TargetKind
	ID           string
	EventID      string // Backwards-compatible alias for ID when Kind == KindEvent
	TargetURL    string
	CanonicalURL string
	ShortURL     string
}

// NormalizeInput normalizes an event ID or URL (full or shortened) into standard URLs.
func NormalizeInput(raw string) (*NormalizedURL, error) {
	norm, err := NormalizeTarget(raw)
	if err != nil {
		return nil, err
	}
	return norm, nil
}

// NormalizeTarget parses and normalizes either an event or artist URL / ID.
func NormalizeTarget(raw string) (*NormalizedURL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("empty input")
	}

	// Direct ID match
	if idRegex.MatchString(trimmed) && !strings.Contains(trimmed, "/") && !strings.Contains(trimmed, ".") {
		return &NormalizedURL{
			Kind:         KindEvent,
			ID:           trimmed,
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
	var id string
	var kind TargetKind = KindEvent
	var targetURL string

	for i, part := range pathParts {
		if part == "artist" {
			if i+1 < len(pathParts) {
				id = pathParts[i+1]
				kind = KindArtist
				targetURL = fmt.Sprintf("https://ticketdive.com/artist/%s", id)
				break
			}
		} else if part == "event" {
			if i+1 < len(pathParts) && pathParts[i+1] == "fc" && i+2 < len(pathParts) {
				id = pathParts[i+2]
				kind = KindEvent
				targetURL = fmt.Sprintf("https://ticketdive.com/event/fc/%s", id)
				break
			} else if i+1 < len(pathParts) {
				id = pathParts[i+1]
				kind = KindEvent
				targetURL = fmt.Sprintf("https://ticketdive.com/event/%s", id)
				break
			}
		}
	}

	if id == "" {
		// Short URL: t-dv.com/<id>
		if u.Host == "t-dv.com" || u.Host == "www.t-dv.com" {
			if len(pathParts) > 0 && pathParts[0] != "" {
				id = pathParts[0]
				kind = KindEvent
				targetURL = fmt.Sprintf("https://ticketdive.com/event/%s", id)
			}
		} else if len(pathParts) > 0 && pathParts[len(pathParts)-1] != "" {
			id = pathParts[len(pathParts)-1]
			targetURL = u.String()
		}
	}

	if id == "" {
		return nil, fmt.Errorf("unable to determine event or artist ID from %q", raw)
	}

	var canonicalURL, shortURL string
	if kind == KindArtist {
		canonicalURL = fmt.Sprintf("https://ticketdive.com/artist/%s", id)
		shortURL = canonicalURL
	} else {
		canonicalURL = fmt.Sprintf("https://ticketdive.com/event/%s", id)
		shortURL = fmt.Sprintf("https://t-dv.com/%s", id)
	}

	return &NormalizedURL{
		Kind:         kind,
		ID:           id,
		EventID:      id,
		TargetURL:    targetURL,
		CanonicalURL: canonicalURL,
		ShortURL:     shortURL,
	}, nil
}
