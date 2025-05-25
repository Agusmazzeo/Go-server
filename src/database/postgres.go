package database

import (
	"context"
	"fmt"
	"server/src/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupDB(cfg *config.Config) (*pgxpool.Pool, error) {
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
		return nil, err
	}

	// Set reasonable pool size for testing
	config.MaxConns = 5
	config.MinConns = 1

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %v\nPlease ensure the database is running and accessible with the provided credentials", err)
	}

	// Test the connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v\nPlease check your database configuration and ensure it's running", err)
	}
	return pool, nil
}
