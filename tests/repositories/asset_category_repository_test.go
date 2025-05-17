package repositories_test

import (
	"context"
	"server/src/models"
	"server/src/repositories"
	"testing"

	"server/tests/repositories/test_init"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssetCategoryRepository(t *testing.T) {
	// Setup test database connection
	db := test_init.SetupTestDB(t)

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
		err := repo.Create(ctx, category)
		require.NoError(t, err)

		// Test GetByID
		retrievedCategory, err := repo.GetByID(ctx, category.ID)
		require.NoError(t, err)
		assert.Equal(t, category.Name, retrievedCategory.Name)
		assert.Equal(t, category.Description, retrievedCategory.Description)
	})

	t.Run("GetAll", func(t *testing.T) {
		ctx := context.Background()

		// Create multiple categories
		categories := []*models.AssetCategory{
			{Name: "Category 1", Description: "Description 1"},
			{Name: "Category 2", Description: "Description 2"},
		}

		for _, category := range categories {
			err := repo.Create(ctx, category)
			require.NoError(t, err)
		}

		// Test GetAll
		retrievedCategories, err := repo.GetAll(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrievedCategories), len(categories))
	})

	t.Run("GetByID for non-existent category", func(t *testing.T) {
		ctx := context.Background()
		nonExistentID := 999999

		category, err := repo.GetByID(ctx, nonExistentID)
		require.Error(t, err)
		assert.Nil(t, category)
	})
}
