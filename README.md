# lumidive

TicketDive (ticketdive.com / t-dv.com) event metadata proxy, CLI, and iCalendar feed generator written in Go.

---

## Overview

lumidive provides high-performance, structured access to TicketDive events, ticket tiers, real-time availability/sold-out status, survey questions, and iCalendar (.ics) exports.

### Key Capabilities

- **URL and Short-Link Resolution**: Accepts full event URLs (`https://ticketdive.com/event/<id>`), official short URLs (`https://t-dv.com/<id>`), FC event URLs (`https://ticketdive.com/event/fc/<id>`), and bare event IDs.
- **Deep Metadata Extraction**: Parses Next.js SSR hydration payloads (`__NEXT_DATA__`) into strongly-typed structures including event summaries, stages, schedules, artists, ticket categories, ticket prices, fees, real-time stock ratios, and questionnaire fields.
- **High Throughput and Protection**: In-memory TTL caching with `singleflight` request coalescing to minimize upstream load.
- **iCalendar (.ics) Feed**: Converts TicketDive schedules into RFC 5545 `.ics` feeds for Google Calendar and Apple Calendar integration.
- **OpenAPI 3.0 Compliant**: Strictly typed REST API generated via `oapi-codegen`.

---

## Installation and Quick Start

### Build from Source

```bash
git clone https://github.com/AobaIwaki123/lumidive.git
cd lumidive
go build -o bin/lumidive ./cmd/lumidive
```

### CLI Usage

```bash
# Parse event metadata and print JSON
./bin/lumidive parse https://ticketdive.com/event/plkt1022

# Parse via short URL or ID
./bin/lumidive parse t-dv.com/plkt1022
./bin/lumidive parse plkt1022

# Generate iCalendar (.ics) feed to stdout
./bin/lumidive ical plkt1022 > event.ics

# Start API server
./bin/lumidive server --port 8080 --cache-ttl 60s
```

---

## API Endpoints

| Method | Path | Description |
|:---|:---|:---|
| `GET` | `/healthz` | Health check |
| `GET` | `/api/v1/events/{eventId}` | Fetch event metadata by ID or slug |
| `GET` | `/api/v1/events?url={url}` | Fetch event metadata by URL query parameter |
| `POST` | `/api/v1/events/parse` | Parse event by JSON payload (`{"url": "..."}` or `{"id": "..."}`) |
| `GET` | `/api/v1/events/{eventId}/ical` | Get iCalendar (.ics) calendar feed |

---

## Local Verification

Run the strict local verification script before pushing or submitting PRs:

```bash
./scripts/verify-all.sh
```

This validates code generation drift, runs `golangci-lint`, executes tests with `-race`, and builds all binaries.

---

## License

MIT License. See [LICENSE](LICENSE) for details.
