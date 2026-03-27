package reader

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache_BasicOperations(t *testing.T) {
	tmpDir := t.TempDir()
	cache, err := NewCache(tmpDir, 300)
	require.NoError(t, err)

	// Cache miss
	entry, content, hit := cache.Get("https://example.com")
	assert.False(t, hit)
	assert.Nil(t, entry)
	assert.Empty(t, content)

	// Set
	testEntry := &CacheEntry{
		URL:         "https://example.com",
		CachedAt:    time.Now(),
		WordCount:   42,
		HTTPStatus:  200,
		ContentType: "text/html",
	}
	testContent := "# Example\n\nHello world"
	err = cache.Set("https://example.com", testEntry, testContent)
	require.NoError(t, err)

	// Cache hit
	entry, content, hit = cache.Get("https://example.com")
	assert.True(t, hit)
	assert.Equal(t, "https://example.com", entry.URL)
	assert.Equal(t, 42, entry.WordCount)
	assert.Equal(t, testContent, content)

	// Verify files exist
	key := cache.Key("https://example.com")
	assert.FileExists(t, filepath.Join(tmpDir, key+".md"))
	assert.FileExists(t, filepath.Join(tmpDir, key+".meta.json"))
}

func TestCache_TTLExpiry(t *testing.T) {
	tmpDir := t.TempDir()
	// TTL of 1 second
	cache, err := NewCache(tmpDir, 1)
	require.NoError(t, err)

	entry := &CacheEntry{
		URL:      "https://example.com",
		CachedAt: time.Now(),
	}
	err = cache.Set("https://example.com", entry, "content")
	require.NoError(t, err)

	// Should hit immediately
	_, _, hit := cache.Get("https://example.com")
	assert.True(t, hit)

	// Wait for expiry
	time.Sleep(1100 * time.Millisecond)

	// Should miss after TTL
	_, _, hit = cache.Get("https://example.com")
	assert.False(t, hit)
}

func TestCache_KeyDeterministic(t *testing.T) {
	tmpDir := t.TempDir()
	cache, err := NewCache(tmpDir, 300)
	require.NoError(t, err)

	key1 := cache.Key("https://example.com/article")
	key2 := cache.Key("https://example.com/article")
	assert.Equal(t, key1, key2, "same URL should produce same key")

	key3 := cache.Key("https://example.com/other")
	assert.NotEqual(t, key1, key3, "different URLs should produce different keys")

	// Key should be 64 hex chars (SHA256)
	assert.Len(t, key1, 64)
}

func TestCache_Invalidate(t *testing.T) {
	tmpDir := t.TempDir()
	cache, err := NewCache(tmpDir, 300)
	require.NoError(t, err)

	cache.Set("https://example.com", &CacheEntry{URL: "https://example.com"}, "content")
	_, _, hit := cache.Get("https://example.com")
	assert.True(t, hit)

	cache.Invalidate("https://example.com")
	_, _, hit = cache.Get("https://example.com")
	assert.False(t, hit)
}

func TestCache_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	cache, err := NewCache(tmpDir, 300)
	require.NoError(t, err)

	cache.Set("https://a.com", &CacheEntry{URL: "https://a.com"}, "a")
	cache.Set("https://b.com", &CacheEntry{URL: "https://b.com"}, "b")

	total, _, err := cache.Stats()
	require.NoError(t, err)
	assert.Equal(t, 2, total)

	cache.Clear()

	total, _, err = cache.Stats()
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}

func TestCache_Stats(t *testing.T) {
	tmpDir := t.TempDir()
	cache, err := NewCache(tmpDir, 300)
	require.NoError(t, err)

	total, expired, err := cache.Stats()
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Equal(t, 0, expired)

	cache.Set("https://example.com", &CacheEntry{URL: "https://example.com", CachedAt: time.Now()}, "content")

	total, expired, err = cache.Stats()
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, 0, expired)
}

func TestCache_CreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "sub", "dir")
	cache, err := NewCache(nestedDir, 300)
	require.NoError(t, err)
	assert.DirExists(t, nestedDir)

	// Should work fine after creating nested dirs
	err = cache.Set("https://example.com", &CacheEntry{URL: "https://example.com"}, "content")
	require.NoError(t, err)
}

func TestCache_CachedAtDefault(t *testing.T) {
	tmpDir := t.TempDir()
	cache, err := NewCache(tmpDir, 300)
	require.NoError(t, err)

	before := time.Now()
	cache.Set("https://example.com", &CacheEntry{URL: "https://example.com"}, "content")
	after := time.Now()

	entry, _, hit := cache.Get("https://example.com")
	assert.True(t, hit)
	assert.True(t, entry.CachedAt.After(before.Add(-time.Second)) || entry.CachedAt.Equal(before))
	assert.True(t, entry.CachedAt.Before(after.Add(time.Second)))
}
