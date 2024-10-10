package utils

import (
	"io"
	"os"
	"sync"
	"time"
)

type Cache[T any] struct {
	value      T
	cachedAt   time.Time
	expiration time.Time
	mutex      sync.RWMutex
}

// NewCache initializes a new cache with an empty value.
func NewCache[T any]() *Cache[T] {
	var zero T
	return &Cache[T]{
		value: zero,
	}
}

// Set sets a new value in the cache with an expiration time.
func (c *Cache[T]) Set(value T, duration time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.value = value
	c.cachedAt = time.Now()
	c.expiration = time.Now().Add(duration)
}

// Get retrieves the cached value, checking if it's valid based on refreshAfter.
func (c *Cache[T]) Get(refreshAfter time.Time) (T, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if time.Now().After(c.expiration) || c.cachedAt.After(refreshAfter) {
		var zero T
		return zero, false
	}
	return c.value, true
}

// Clear removes the cached value.
func (c *Cache[T]) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var zero T
	c.value = zero
	c.expiration = time.Time{}
}

// SaveResponseToFile is a middleware that saves the response bytes to a file and returns them
func SaveResponseToFile(reader io.ReadCloser, filePath string) ([]byte, error) {
	// Read all the bytes from the response body
	responseBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	// Save the response bytes to the file
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	_, err = file.Write(responseBytes)
	if err != nil {
		return nil, err
	}

	// Return the response bytes so the calling function can use them
	return responseBytes, nil
}

// ReadResponseFromFile reads a JSON response from a file and returns the content as a byte slice.
func ReadResponseFromFile(filePath string) ([]byte, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read the file content
	responseData, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return responseData, nil
}
