package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type CacheHandlerI interface {
	Set(key string, value interface{}, expiration time.Duration) error
	Get(key string, result interface{}) error
	Delete(key string) error
	Exists(key string) (bool, error)
}

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

func SaveStructToJSONFile(data interface{}, filename string) error {
	// Marshal the struct into JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal struct: %v", err)
	}

	// Create or open the file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// Write JSON data to the file
	_, err = file.Write(jsonData)
	if err != nil {
		return fmt.Errorf("failed to write data to file: %v", err)
	}

	return nil
}

// LoadStructFromJSONFile loads JSON data from a file into a struct.
func LoadStructFromJSONFile(filename string, data interface{}) error {
	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Read the file content
	bytes, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// Unmarshal the JSON data into the provided struct
	err = json.Unmarshal(bytes, data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json: %v", err)
	}

	return nil
}
