package reader

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"os"
	"path/filepath"
	"time"

	"github.com/vanzheng/web-tools/internal/config"
)

// Cache provides local file-based caching for reader results.
type Cache struct {
	dir string
	ttl time.Duration
}

// CacheEntry holds metadata about a cached result.
type CacheEntry struct {
	URL         string    `json:"url"`
	CachedAt    time.Time `json:"cached_at"`
	WordCount   int       `json:"word_count"`
	HTTPStatus  int       `json:"http_status"`
	ContentType string    `json:"content_type"`
}

// NewCache creates a new Cache instance.
func NewCache(dir string, ttlSeconds int) (*Cache, error) {
	if ttlSeconds <= 0 {
		ttlSeconds = config.DefaultCacheTTL
	}
	// Ensure cache directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create cache dir %s: %w", dir, err)
	}
	return &Cache{
		dir: dir,
		ttl: time.Duration(ttlSeconds) * time.Second,
	}, nil
}

// Key generates a SHA256 hash for a URL or file path.
func (c *Cache) Key(source string) string {
	h := sha256.Sum256([]byte(source))
	return hex.EncodeToString(h[:])
}

// contentPath returns the .md file path for a cache key.
func (c *Cache) contentPath(key string) string {
	return filepath.Join(c.dir, key+".md")
}

// metaPath returns the .meta.json file path for a cache key.
func (c *Cache) metaPath(key string) string {
	return filepath.Join(c.dir, key+".meta.json")
}

// Get retrieves a cached result. Returns (entry, content, hit).
func (c *Cache) Get(source string) (*CacheEntry, string, bool) {
	key := c.Key(source)

	// Read metadata
	metaData, err := os.ReadFile(c.metaPath(key))
	if err != nil {
		return nil, "", false
	}

	var entry CacheEntry
	if err := json.Unmarshal(metaData, &entry); err != nil {
		return nil, "", false
	}

	// Check TTL
	if time.Since(entry.CachedAt) > c.ttl {
		// Expired, clean up
		os.Remove(c.contentPath(key))
		os.Remove(c.metaPath(key))
		return nil, "", false
	}

	// Read content
	content, err := os.ReadFile(c.contentPath(key))
	if err != nil {
		return nil, "", false
	}

	return &entry, string(content), true
}

// Set stores a result in the cache.
func (c *Cache) Set(source string, entry *CacheEntry, content string) error {
	key := c.Key(source)

	// Set default CachedAt if not provided
	if entry.CachedAt.IsZero() {
		entry.CachedAt = time.Now()
	}

	// Write metadata
	metaData, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal cache metadata: %w", err)
	}
	if err := os.WriteFile(c.metaPath(key), metaData, 0644); err != nil {
		return fmt.Errorf("cannot write cache metadata: %w", err)
	}

	// Write content
	if err := os.WriteFile(c.contentPath(key), []byte(content), 0644); err != nil {
		return fmt.Errorf("cannot write cache content: %w", err)
	}

	return nil
}

// Invalidate removes a specific cache entry.
func (c *Cache) Invalidate(source string) error {
	key := c.Key(source)
	os.Remove(c.contentPath(key))
	os.Remove(c.metaPath(key))
	return nil
}

// Clear removes all cache entries.
func (c *Cache) Clear() error {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		os.Remove(filepath.Join(c.dir, e.Name()))
	}
	return nil
}

// Stats returns the number of cached entries.
func (c *Cache) Stats() (total int, expired int, err error) {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return 0, 0, err
	}

	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".meta.json") {
			total++
			metaData, err := os.ReadFile(filepath.Join(c.dir, e.Name()))
			if err != nil {
				continue
			}
			var entry CacheEntry
			if json.Unmarshal(metaData, &entry) == nil {
				if time.Since(entry.CachedAt) > c.ttl {
					expired++
				}
			}
		}
	}
	return total, expired, nil
}
