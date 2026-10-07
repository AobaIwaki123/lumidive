// Package ticketdive provides client and parser capabilities for TicketDive events.
package ticketdive

import (
	"context"
	"sync"
	"time"

	"github.com/AobaIwaki123/lumidive/pkg/api"
	"golang.org/x/sync/singleflight"
)

type cacheEntry struct {
	cachedAt time.Time
	event    *api.Event
}

// CachedService wraps Client with an in-memory TTL cache and singleflight request deduplication.
type CachedService struct {
	client *Client
	ttl    time.Duration
	mu     sync.RWMutex
	cache  map[string]cacheEntry
	sf     singleflight.Group
}

// NewCachedService creates a new CachedService.
func NewCachedService(client *Client, ttl time.Duration) *CachedService {
	return &CachedService{
		client: client,
		ttl:    ttl,
		cache:  make(map[string]cacheEntry),
	}
}

// GetEvent fetches an event with caching.
// Returns (event, cached, error).
func (s *CachedService) GetEvent(ctx context.Context, rawInput string) (*api.Event, bool, error) {
	norm, err := NormalizeInput(rawInput)
	if err != nil {
		return nil, false, err
	}

	key := norm.EventID

	// Check cache
	s.mu.RLock()
	entry, found := s.cache[key]
	s.mu.RUnlock()

	if found && time.Since(entry.cachedAt) < s.ttl {
		return entry.event, true, nil
	}

	// Singleflight fetch
	res, err, _ := s.sf.Do(key, func() (any, error) {
		// Double check under lock in case another goroutine just resolved it
		s.mu.RLock()
		e, ok := s.cache[key]
		s.mu.RUnlock()
		if ok && time.Since(e.cachedAt) < s.ttl {
			return e.event, nil
		}

		fetched, fetchErr := s.client.FetchEvent(ctx, rawInput)
		if fetchErr != nil {
			return nil, fetchErr
		}

		s.mu.Lock()
		s.cache[key] = cacheEntry{
			cachedAt: time.Now(),
			event:    fetched,
		}
		s.mu.Unlock()

		return fetched, nil
	})

	if err != nil {
		return nil, false, err
	}

	ev, ok := res.(*api.Event)
	if !ok {
		return nil, false, nil
	}

	return ev, false, nil
}
