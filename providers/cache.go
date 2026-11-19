package providers

import (
	"errors"
	"net/http"
	"sync"
	"time"
)

// FetchTimeout bounds one download of a provider's IP list.
const FetchTimeout = 10 * time.Second

// HTTPClient downloads the provider lists. Upstream used http.Get with no timeout.
var HTTPClient = &http.Client{Timeout: FetchTimeout}

// ErrNoList means no list was ever fetched and the last attempt failed.
var ErrNoList = errors.New("provider IP list not fetched yet")

// ListCache keeps one provider IP list for the whole Traefik process.
//
// Traefik builds a middleware instance for every router and entry point on every
// configuration change, and upstream fetched the lists inside each instance's
// constructor: with 19 routes on two entry points a change took ~80 s to apply.
// Here the first instance fetches, the others reuse the result until it is TTL old.
// A failed refresh keeps serving the last good list; without one the caller falls
// back as upstream did, and the fetch is not retried before RetryAfter.
type ListCache struct {
	Fetch      func() ([]string, error)
	TTL        time.Duration
	RetryAfter time.Duration

	mu        sync.Mutex
	list      []string
	fetchedAt time.Time
	failedAt  time.Time
}

// Get returns a copy of the cached list, fetching it when it is missing or stale.
func (c *ListCache) Get(now time.Time) ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.list) > 0 && now.Sub(c.fetchedAt) < c.TTL {
		return c.copyList(), nil
	}
	if !c.failedAt.IsZero() && now.Sub(c.failedAt) < c.RetryAfter {
		return c.stale()
	}

	list, err := c.Fetch()
	if err != nil || len(list) == 0 {
		c.failedAt = now
		return c.stale()
	}
	c.list = list
	c.fetchedAt = now
	c.failedAt = time.Time{}
	return c.copyList(), nil
}

func (c *ListCache) stale() ([]string, error) {
	if len(c.list) == 0 {
		return nil, ErrNoList
	}
	return c.copyList(), nil
}

func (c *ListCache) copyList() []string {
	return append([]string(nil), c.list...)
}
