package utils_test

import (
	"server/src/utils"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	t.Run("should return the cached string value if valid", func(t *testing.T) {
		cache := utils.NewCache[string]()
		cache.Set("test value", 1*time.Minute)

		value, found := cache.Get(time.Now())
		if !found || value != "test value" {
			t.Error("expected 'test value', got", value)
		}
	})

	t.Run("should return a zero value if the cache is expired", func(t *testing.T) {
		cache := utils.NewCache[string]()
		cache.Set("test value", 1*time.Second)
		time.Sleep(2 * time.Second)

		value, found := cache.Get(time.Now())
		if found {
			t.Error("expected cache miss, got", value)
		}
	})

	t.Run("should return a zero value if the cache is older than refreshAfter", func(t *testing.T) {
		cache := utils.NewCache[string]()
		cache.Set("test value", 1*time.Minute)

		refreshAfter := time.Now().Add(-5 * time.Minute)
		value, found := cache.Get(refreshAfter)
		if found {
			t.Error("expected cache miss due to refreshAfter, got", value)
		}
	})

	t.Run("should return the cached struct value if valid", func(t *testing.T) {
		type User struct {
			Name  string
			Email string
		}
		cache := utils.NewCache[User]()
		user := User{Name: "John Doe", Email: "john@example.com"}
		cache.Set(user, 1*time.Minute)

		value, found := cache.Get(time.Now())
		if !found || value.Name != "John Doe" {
			t.Errorf("expected 'John Doe', got %+v", value)
		}
	})
}
