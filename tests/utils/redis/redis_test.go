package redis_test

import (
	"log"
	"os"
	"server/src/config"
	redis "server/src/utils/redis"
	"testing"
	"time"
)

type SampleData struct {
	Name  string
	Age   int
	Email string
}

func TestRedisHandler(t *testing.T) {
	cfg, err := config.LoadConfig("../../../settings", os.Getenv("ENV"))
	if err != nil {
		log.Println(err, "Error while loading config")
		os.Exit(1)
	}
	handler, err := redis.NewRedisHandler(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize Redis handler: %v", err)
	}
	defer handler.Close()

	key := "test_key"
	expiration := 10 * time.Second

	// Test Set and Get with a string
	t.Run("Set and Get with string", func(t *testing.T) {
		value := "test_value"
		err := handler.Set(key, value, expiration)
		if err != nil {
			t.Fatalf("Failed to set key in Redis: %v", err)
		}

		var gotValue string
		err = handler.Get(key, &gotValue)
		if err != nil {
			t.Fatalf("Failed to get key from Redis: %v", err)
		}
		if gotValue != value {
			t.Errorf("Value mismatch: got %v, want %v", gotValue, value)
		}
	})

	// Test Set and Get with a struct
	t.Run("Set and Get with struct", func(t *testing.T) {
		value := SampleData{Name: "John Doe", Age: 30, Email: "john.doe@example.com"}
		err := handler.Set(key, value, expiration)
		if err != nil {
			t.Fatalf("Failed to set key in Redis: %v", err)
		}

		var gotValue SampleData
		err = handler.Get(key, &gotValue)
		if err != nil {
			t.Fatalf("Failed to get key from Redis: %v", err)
		}
		if gotValue != value {
			t.Errorf("Struct mismatch: got %+v, want %+v", gotValue, value)
		}
	})

	// Test Exists
	t.Run("Exists", func(t *testing.T) {
		exists, err := handler.Exists(key)
		if err != nil {
			t.Fatalf("Failed to check key existence: %v", err)
		}
		if !exists {
			t.Errorf("Expected key to exist but it does not")
		}
	})

	// Test Expiration
	t.Run("Expiration", func(t *testing.T) {
		time.Sleep(expiration + 1*time.Second) // Wait for the key to expire

		exists, err := handler.Exists(key)
		if err != nil {
			t.Fatalf("Failed to check key existence after expiration: %v", err)
		}
		if exists {
			t.Errorf("Expected key to not exist after expiration but it does")
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		value := "temp_value"
		err := handler.Set(key, value, expiration)
		if err != nil {
			t.Fatalf("Failed to reset key in Redis: %v", err)
		}

		err = handler.Delete(key)
		if err != nil {
			t.Fatalf("Failed to delete key in Redis: %v", err)
		}

		exists, err := handler.Exists(key)
		if err != nil {
			t.Fatalf("Failed to check key existence after deletion: %v", err)
		}
		if exists {
			t.Errorf("Expected key to not exist after deletion but it does")
		}
	})

	// Test Get Non-Existent Key
	t.Run("Get Non-Existent Key", func(t *testing.T) {
		var gotValue string
		err := handler.Get("non_existent_key", &gotValue)
		if err == nil {
			t.Errorf("Expected an error for non-existent key but got none")
		} else if err.Error() != "key does not exist: non_existent_key" {
			t.Errorf("Unexpected error message: got %v", err)
		}
	})

	// Test GenerateUUID with same inputs
	t.Run("Generate UUID with same inputs", func(t *testing.T) {
		inputs := []string{"example", "real", "uuid"}
		uuid1, err := redis.GenerateUUID(inputs...)
		if err != nil {
			t.Fatalf("Failed to generate UUID: %v", err)
		}

		uuid2, err := redis.GenerateUUID(inputs...)
		if err != nil {
			t.Fatalf("Failed to generate UUID on second attempt: %v", err)
		}

		if uuid1 != uuid2 {
			t.Errorf("UUIDs do not match for the same inputs: %s != %s", uuid1, uuid2)
		}
	})

	// Test GenerateUUID with different inputs
	t.Run("Generate UUID with different inputs", func(t *testing.T) {
		inputs1 := []string{"example", "real", "uuid"}
		inputs2 := []string{"different", "input", "test"}

		uuid1, err := redis.GenerateUUID(inputs1...)
		if err != nil {
			t.Fatalf("Failed to generate UUID for first set of inputs: %v", err)
		}

		uuid2, err := redis.GenerateUUID(inputs2...)
		if err != nil {
			t.Fatalf("Failed to generate UUID for second set of inputs: %v", err)
		}

		if uuid1 == uuid2 {
			t.Errorf("UUIDs should not match for different inputs: %s == %s", uuid1, uuid2)
		}
	})
}
