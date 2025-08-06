package repositories_test

import (
	"context"
	"fmt"
	"server/src/models"
	"server/src/repositories"
	"strings"
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

	// Cleanup after test
	defer func() {
		init_test.CleanupTestDataByClientID(t, db, "test-client")
		init_test.CleanupTestDataByAssetName(t, db, "Test Asset")
		init_test.CleanupTestDataByAssetCategoryName(t, db, "Test Category")
	}()

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

	})

	t.Run("Create and GetByName", func(t *testing.T) {
		ctx := context.Background()
		category := &models.AssetCategory{
			Name:        "Test Category",
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
		init_test.CleanupTestDataByClientID(t, db, "test-client")
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
		init_test.CleanupTestDataByClientID(t, db, "test-client")
	})

	t.Run("GetByID for non-existent category", func(t *testing.T) {
		ctx := context.Background()
		nonExistentID := 999999

		category, err := repo.GetByID(ctx, nonExistentID)
		require.Nil(t, err)
		assert.Nil(t, category)

		// Cleanup after this subtest
		init_test.CleanupTestDataByClientID(t, db, "test-client")
	})
}

func TestAssetCategoryRepository_CreateBatch(t *testing.T) {
	// Setup test database connection
	db := init_test.SetupTestDB(t)

	// Create repository instance
	repo := repositories.NewAssetCategoryRepository(db)

	ctx := context.Background()

	// Cleanup test data after test
	defer func() {
		init_test.CleanupTestDataByAssetCategoryName(t, db, "Batch Category 1")
		init_test.CleanupTestDataByAssetCategoryName(t, db, "Batch Category 2")
		init_test.CleanupTestDataByAssetCategoryName(t, db, "Batch Category 3")
		init_test.CleanupTestDataByAssetCategoryName(t, db, "Large Batch Category 1")
		init_test.CleanupTestDataByAssetCategoryName(t, db, "Large Batch Category 50")
		init_test.CleanupTestDataByAssetCategoryName(t, db, "TX Category")
	}()

	t.Run("CreateBatch with multiple categories", func(t *testing.T) {
		// Create batch of categories
		categories := []models.AssetCategory{
			{
				Name:        "Batch Category 1",
				Description: "First batch category",
			},
			{
				Name:        "Batch Category 2",
				Description: "Second batch category",
			},
			{
				Name:        "Batch Category 3",
				Description: "Third batch category",
			},
		}

		// Execute batch create
		err := repo.CreateBatch(ctx, categories, nil)
		require.NoError(t, err)

		// Verify all categories were created
		for _, expectedCategory := range categories {
			category, err := repo.GetByName(ctx, expectedCategory.Name)
			require.NoError(t, err)
			assert.NotNil(t, category)
			assert.Equal(t, expectedCategory.Name, category.Name)
			assert.Equal(t, expectedCategory.Description, category.Description)
			assert.NotZero(t, category.ID)
		}
	})

	t.Run("CreateBatch with conflict resolution", func(t *testing.T) {
		// First, create a category
		initialCategories := []models.AssetCategory{
			{
				Name:        "Batch Category 1",
				Description: "Original description",
			},
		}

		err := repo.CreateBatch(ctx, initialCategories, nil)
		require.NoError(t, err)

		// Get the original category to check ID
		originalCategory, err := repo.GetByName(ctx, "Batch Category 1")
		require.NoError(t, err)
		require.NotNil(t, originalCategory)

		// Now create batch with same name but different description (should update)
		conflictCategories := []models.AssetCategory{
			{
				Name:        "Batch Category 1",    // Same name
				Description: "Updated description", // Different description
			},
		}

		err = repo.CreateBatch(ctx, conflictCategories, nil)
		require.NoError(t, err)

		// Verify the category was updated, not duplicated
		updatedCategory, err := repo.GetByName(ctx, "Batch Category 1")
		require.NoError(t, err)
		assert.NotNil(t, updatedCategory)
		assert.Equal(t, originalCategory.ID, updatedCategory.ID)            // Same ID
		assert.Equal(t, "Updated description", updatedCategory.Description) // Updated description
	})

	t.Run("CreateBatch with empty array", func(t *testing.T) {
		// Should handle empty array gracefully
		err := repo.CreateBatch(ctx, []models.AssetCategory{}, nil)
		require.NoError(t, err)
	})

	t.Run("CreateBatch with transaction context", func(t *testing.T) {
		// Test with explicit transaction
		tx, err := db.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		categories := []models.AssetCategory{
			{
				Name:        "TX Category",
				Description: "Category created in transaction",
			},
		}

		err = repo.CreateBatch(ctx, categories, tx)
		require.NoError(t, err)

		// Commit the transaction
		err = tx.Commit(ctx)
		require.NoError(t, err)

		// Verify the category was created
		category, err := repo.GetByName(ctx, "TX Category")
		require.NoError(t, err)
		assert.NotNil(t, category)
		assert.Equal(t, "TX Category", category.Name)
		assert.Equal(t, "Category created in transaction", category.Description)
	})

	t.Run("CreateBatch with large batch", func(t *testing.T) {
		// Test with a larger batch to verify performance
		batchSize := 50
		categories := make([]models.AssetCategory, batchSize)

		for i := 0; i < batchSize; i++ {
			categories[i] = models.AssetCategory{
				Name:        fmt.Sprintf("Large Batch Category %d", i+1),
				Description: fmt.Sprintf("Description for category %d", i+1),
			}
		}

		// Execute batch create
		err := repo.CreateBatch(ctx, categories, nil)
		require.NoError(t, err)

		// Verify some sample categories were created
		firstCategory, err := repo.GetByName(ctx, "Large Batch Category 1")
		require.NoError(t, err)
		assert.NotNil(t, firstCategory)
		assert.Equal(t, "Description for category 1", firstCategory.Description)

		lastCategory, err := repo.GetByName(ctx, "Large Batch Category 50")
		require.NoError(t, err)
		assert.NotNil(t, lastCategory)
		assert.Equal(t, "Description for category 50", lastCategory.Description)

		// Verify all categories exist
		allCategories, err := repo.GetAll(ctx)
		require.NoError(t, err)

		// Count how many of our test categories exist
		testCategoryCount := 0
		for _, category := range allCategories {
			if strings.Contains(category.Name, "Large Batch Category") {
				testCategoryCount++
			}
		}
		assert.Equal(t, batchSize, testCategoryCount)
	})
}
