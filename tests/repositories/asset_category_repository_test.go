package repositories_test

import (
	"context"
	"server/src/models"
	"server/src/repositories"
	"testing"

	"server/tests/init_test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssetCategoryRepository(t *testing.T) {
	// Setup test database connection
	db := init_test.SetupTestDB(t)

	// Create repository instance
	repo := repositories.NewAssetCategoryRepository(db)

	// Test cases
	t.Run("Create and GetByID", func(t *testing.T) {
		ctx := context.Background()
		category := &models.AssetCategory{
			Name:        "Test Category",
			Description: "Test Description",
		}

		// Test Create
		err := repo.Create(ctx, category, nil)
		require.NoError(t, err)

		// Test GetByID
		retrievedCategory, err := repo.GetByID(ctx, category.ID)
		require.NoError(t, err)
		assert.Equal(t, category.Name, retrievedCategory.Name)
		assert.Equal(t, category.Description, retrievedCategory.Description)

		// Cleanup after this subtest
		init_test.CleanupTestData(t, db, "test-client")
	})

	t.Run("Create and GetByName", func(t *testing.T) {
		ctx := context.Background()
		category := &models.AssetCategory{
			Name:        "Test Category By Name",
			Description: "Test Description",
		}

		// Test Create
		err := repo.Create(ctx, category, nil)
		require.NoError(t, err)

		// Test GetByName
		retrievedCategory, err := repo.GetByName(ctx, category.Name)
		require.NoError(t, err)
		assert.Equal(t, category.Name, retrievedCategory.Name)
		assert.Equal(t, category.Description, retrievedCategory.Description)

		// Cleanup after this subtest
		init_test.CleanupTestData(t, db, "test-client")
	})

	t.Run("GetAll", func(t *testing.T) {
		ctx := context.Background()

		// Create multiple categories
		categories := []*models.AssetCategory{
			{Name: "Category 1", Description: "Description 1"},
			{Name: "Category 2", Description: "Description 2"},
		}

		for _, category := range categories {
			err := repo.Create(ctx, category, nil)
			require.NoError(t, err)
		}

		// Test GetAll
		retrievedCategories, err := repo.GetAll(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrievedCategories), len(categories))

		// Cleanup after this subtest
		init_test.CleanupTestData(t, db, "test-client")
	})

	t.Run("GetByID for non-existent category", func(t *testing.T) {
		ctx := context.Background()
		nonExistentID := 999999

		category, err := repo.GetByID(ctx, nonExistentID)
		require.Nil(t, err)
		assert.Nil(t, category)

		// Cleanup after this subtest
		init_test.CleanupTestData(t, db, "test-client")
	})
}
