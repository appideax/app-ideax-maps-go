package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemoryCache_SetGetDelete(t *testing.T) {
	c := NewMemoryCache(1024)

	c.Set("a", []byte("uno"), time.Minute)
	value, ok := c.Get("a")
	assert.True(t, ok)
	assert.Equal(t, "uno", string(value))

	c.Delete("a")
	_, ok = c.Get("a")
	assert.False(t, ok)
}

func TestMemoryCache_Expires(t *testing.T) {
	c := NewMemoryCache(1024)

	c.Set("a", []byte("uno"), time.Nanosecond)
	time.Sleep(time.Millisecond)

	_, ok := c.Get("a")
	assert.False(t, ok)
}

func TestMemoryCache_EvictsLeastRecentlyUsed(t *testing.T) {
	c := NewMemoryCache(10)

	c.Set("a", []byte("12345"), time.Minute)
	c.Set("b", []byte("12345"), time.Minute)
	_, _ = c.Get("a")
	c.Set("c", []byte("12345"), time.Minute)

	_, okA := c.Get("a")
	_, okB := c.Get("b")
	_, okC := c.Get("c")
	assert.True(t, okA)
	assert.False(t, okB)
	assert.True(t, okC)
}

func TestMemoryCache_SkipsValuesOverBudget(t *testing.T) {
	c := NewMemoryCache(4)

	c.Set("a", []byte("12345"), time.Minute)

	_, ok := c.Get("a")
	assert.False(t, ok)
}
