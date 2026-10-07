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
	result   *ParseResult
}

type artistCacheEntry struct {
	cachedAt time.Time
	result   *ArtistParseResult
}

// CachedService wraps Client with an in-memory TTL cache and singleflight request deduplication.
type CachedService struct {
	client      *Client
	ttl         time.Duration
	mu          sync.RWMutex
	cache       map[string]cacheEntry
	artistCache map[string]artistCacheEntry
	sf          singleflight.Group
}

// NewCachedService creates a new CachedService.
func NewCachedService(client *Client, ttl time.Duration) *CachedService {
	return &CachedService{
		client:      client,
		ttl:         ttl,
		cache:       make(map[string]cacheEntry),
		artistCache: make(map[string]artistCacheEntry),
	}
}

// GetEvent fetches an event with caching.
// Returns (event, cached, source, cachedAt, error).
func (s *CachedService) GetEvent(ctx context.Context, rawInput string, refresh bool) (*api.Event, bool, *api.SourceMetadata, *time.Time, error) {
	norm, err := NormalizeInput(rawInput)
	if err != nil {
		return nil, false, nil, nil, err
	}

	key := norm.EventID

	// Check cache if not refresh
	if !refresh {
		s.mu.RLock()
		entry, found := s.cache[key]
		s.mu.RUnlock()

		if found && time.Since(entry.cachedAt) < s.ttl {
			return entry.result.Event, true, entry.result.Source, &entry.cachedAt, nil
		}
	}

	// Singleflight fetch
	res, err, _ := s.sf.Do(key, func() (any, error) {
		if !refresh {
			s.mu.RLock()
			e, ok := s.cache[key]
			s.mu.RUnlock()
			if ok && time.Since(e.cachedAt) < s.ttl {
				return e, nil
			}
		}

		fetched, fetchErr := s.client.FetchEvent(ctx, rawInput)
		if fetchErr != nil {
			return nil, fetchErr
		}

		entry := cacheEntry{
			cachedAt: time.Now().UTC(),
			result:   fetched,
		}

		s.mu.Lock()
		s.cache[key] = entry
		s.mu.Unlock()

		return entry, nil
	})

	if err != nil {
		return nil, false, nil, nil, err
	}

	entry, ok := res.(cacheEntry)
	if !ok || entry.result == nil {
		return nil, false, nil, nil, nil
	}

	return entry.result.Event, false, entry.result.Source, &entry.cachedAt, nil
}

// GetArtist fetches an artist profile and events with caching.
// Returns (artist, cached, source, cachedAt, error).
func (s *CachedService) GetArtist(ctx context.Context, rawInput string, refresh bool) (*api.ArtistDetail, bool, *api.SourceMetadata, *time.Time, error) {
	norm, err := NormalizeTarget(rawInput)
	if err != nil {
		return nil, false, nil, nil, err
	}

	key := "artist:" + norm.ID

	// Check cache if not refresh
	if !refresh {
		s.mu.RLock()
		entry, found := s.artistCache[key]
		s.mu.RUnlock()

		if found && time.Since(entry.cachedAt) < s.ttl {
			return entry.result.Artist, true, entry.result.Source, &entry.cachedAt, nil
		}
	}

	// Singleflight fetch
	res, err, _ := s.sf.Do(key, func() (any, error) {
		if !refresh {
			s.mu.RLock()
			e, ok := s.artistCache[key]
			s.mu.RUnlock()
			if ok && time.Since(e.cachedAt) < s.ttl {
				return e, nil
			}
		}

		fetched, fetchErr := s.client.FetchArtist(ctx, rawInput)
		if fetchErr != nil {
			return nil, fetchErr
		}

		entry := artistCacheEntry{
			cachedAt: time.Now().UTC(),
			result:   fetched,
		}

		s.mu.Lock()
		s.artistCache[key] = entry
		s.mu.Unlock()

		return entry, nil
	})

	if err != nil {
		return nil, false, nil, nil, err
	}

	entry, ok := res.(artistCacheEntry)
	if !ok || entry.result == nil {
		return nil, false, nil, nil, nil
	}

	return entry.result.Artist, false, entry.result.Source, &entry.cachedAt, nil
}
