package init_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"server/src/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	TestDB *pgxpool.Pool
)

// SetupTestDB initializes the test database connection and verifies tables exist
func SetupTestDB(t *testing.T) *pgxpool.Pool {
	if TestDB != nil {
		return TestDB
	}

	// Load test configuration
	cfg, err := loadTestConfig()
	if err != nil {
		t.Fatalf("Failed to load test configuration: %v", err)
	}

	dsn := cfg.Databases.SQL.ConnectionString
	if dsn == "" {
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			cfg.Databases.SQL.Host,
			cfg.Databases.SQL.Username,
			cfg.Databases.SQL.Password,
			cfg.Databases.SQL.Database,
			cfg.Databases.SQL.Port)
	}

	// Create connection pool
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("Failed to parse database config: %v", err)
	}

	// Set reasonable pool size for testing
	config.MaxConns = 5
	config.MinConns = 1

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v\nPlease ensure the database is running and accessible with the provided credentials.", err)
	}

	// Test the connection
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("Failed to ping database: %v\nPlease check your database configuration and ensure it's running.", err)
	}

	TestDB = pool
	return pool
}

// loadTestConfig loads the test configuration from appsettings.TESTING.yaml
func loadTestConfig() (*config.Config, error) {
	// Get the service root path (where go.mod is located)
	serviceRoot, err := getServiceRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get service root path: %w", err)
	}

	// Load the configuration using the settings directory in the service root
	cfg, err := config.LoadConfig(filepath.Join(serviceRoot, "settings"), "TESTING")
	if err != nil {
		return nil, fmt.Errorf("failed to load test configuration: %w", err)
	}

	return cfg, nil
}

// getServiceRoot returns the absolute path to the service root directory
func getServiceRoot() (string, error) {
	// Start from the current working directory
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree until we find go.mod
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd, nil
		}

		// Move up one directory
		parent := filepath.Dir(wd)
		if parent == wd {
			return "", fmt.Errorf("go.mod not found in any parent directory")
		}
		wd = parent
	}
}

// CleanupTestDB closes the database connection
func CleanupTestDB() {
	if TestDB != nil {
		TestDB.Close()
		TestDB = nil
	}
}

// TruncateTables truncates all tables in the test database
func TruncateTables(t *testing.T, pool *pgxpool.Pool) {
	if pool == nil {
		t.Fatal("Database connection not initialized")
	}

	tables := []string{
		"sync_logs",
		"asset_categories",
		"transactions",
		"assets",
		"holdings",
	}

	for _, table := range tables {
		_, err := pool.Exec(context.Background(), fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Fatalf("Failed to truncate table %s: %v", table, err)
		}
	}
}

// CleanupTestData deletes specific test data by client ID or other identifiers
// This allows each test to clean up only its own data
func CleanupTestData(t *testing.T, pool *pgxpool.Pool, clientID string) {
	if pool == nil {
		t.Fatal("Database connection not initialized")
	}

	ctx := context.Background()

	// Delete in reverse order of dependencies to avoid foreign key constraints
	queries := []string{
		fmt.Sprintf("DELETE FROM transactions WHERE client_id = $1"),
		fmt.Sprintf("DELETE FROM holdings WHERE client_id = $1"),
		fmt.Sprintf("DELETE FROM assets WHERE external_id LIKE $1"),
		fmt.Sprintf("DELETE FROM asset_categories WHERE name LIKE $1"),
	}

	for _, query := range queries {
		var err error
		if query == "DELETE FROM assets WHERE external_id LIKE $1" {
			_, err = pool.Exec(ctx, query, "EXT-%")
		} else if query == "DELETE FROM asset_categories WHERE name LIKE $1" {
			// More specific cleanup for categories - only delete exact matches
			_, err = pool.Exec(ctx, "DELETE FROM asset_categories WHERE name IN ('Test Category', 'Test Category By Name', 'Category 1', 'Category 2')")
		} else {
			_, err = pool.Exec(ctx, query, clientID)
		}
		if err != nil {
			t.Logf("Warning: Failed to cleanup test data with query '%s': %v", query, err)
		}
	}
}

// CleanupAllTestData removes all test data from all tables
// Use this only when you need to clean up everything
func CleanupAllTestData(t *testing.T, pool *pgxpool.Pool) {
	if pool == nil {
		t.Fatal("Database connection not initialized")
	}

	ctx := context.Background()

	// Delete in reverse order of dependencies
	tables := []string{
		"sync_logs",
		"transactions",
		"holdings",
		"assets",
		"asset_categories",
	}

	for _, table := range tables {
		_, err := pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Logf("Warning: Failed to delete from table %s: %v", table, err)
		}
	}
}
