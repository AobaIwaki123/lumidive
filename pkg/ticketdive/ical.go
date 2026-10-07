// Package ticketdive provides client and parser capabilities for TicketDive events.
package ticketdive

import (
	"fmt"
	"strings"
	"time"

	"github.com/AobaIwaki123/lumidive/pkg/api"
)

// GenerateICal creates an iCalendar (RFC 5545) representation of an event.
func GenerateICal(event *api.Event) (string, error) {
	if event == nil {
		return "", fmt.Errorf("event is nil")
	}

	var sb strings.Builder
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//AobaIwaki123//lumidive//EN\r\n")
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("METHOD:PUBLISH\r\n")

	sb.WriteString("BEGIN:VEVENT\r\n")
	sb.WriteString(fmt.Sprintf("UID:%s@lumidive.aooba.net\r\n", event.Id))
	sb.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", time.Now().UTC().Format("20060102T150405Z")))

	// Parse start and end times from stages if available
	var start time.Time
	var end time.Time
	var location string

	if event.Stages != nil && len(*event.Stages) > 0 {
		stage := (*event.Stages)[0]
		if stage.StartAt != nil && *stage.StartAt != "" {
			t, err := time.Parse(time.RFC3339, *stage.StartAt)
			if err == nil {
				start = t
			}
		}
		if stage.Venue != nil && stage.Venue.Name != nil {
			location = *stage.Venue.Name
		}
	}

	if start.IsZero() {
		start = time.Now().UTC()
	}
	end = start.Add(3 * time.Hour) // default duration 3 hours

	sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", start.UTC().Format("20060102T150405Z")))
	sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", end.UTC().Format("20060102T150405Z")))

	// Escape summary and description
	summary := escapeICalText(event.Name)
	sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", summary))

	if location != "" {
		sb.WriteString(fmt.Sprintf("LOCATION:%s\r\n", escapeICalText(location)))
	}

	description := fmt.Sprintf("URL: %s\n\n", event.Url)
	if event.Detail != nil {
		description += *event.Detail
	}
	sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICalText(description)))
	sb.WriteString(fmt.Sprintf("URL:%s\r\n", event.Url))

	sb.WriteString("STATUS:CONFIRMED\r\n")
	sb.WriteString("END:VEVENT\r\n")
	sb.WriteString("END:VCALENDAR\r\n")

	return sb.String(), nil
}

func escapeICalText(s string) string {
	r := strings.ReplaceAll(s, "\\", "\\\\")
	r = strings.ReplaceAll(r, ";", "\\;")
	r = strings.ReplaceAll(r, ",", "\\,")
	r = strings.ReplaceAll(r, "\r\n", "\\n")
	r = strings.ReplaceAll(r, "\n", "\\n")
	return r
}
