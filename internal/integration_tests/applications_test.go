package integration_tests

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/authorizerdev/authorizer/internal/graph/model"
	"github.com/authorizerdev/authorizer/internal/storage/schemas"
)

// TestApplications tests the M2M Application CRUD operations at the storage layer
func TestApplications(t *testing.T) {
	cfg := getTestConfig()
	ts := initTestSetup(t, cfg)

	// Use background context for storage calls
	storageCtx := context.Background()

	t.Run("should create application", func(t *testing.T) {
		application := &schemas.Application{
			Name:         "test-m2m-app-" + uuid.New().String(),
			Description:  "Test M2M application",
			ClientID:     uuid.New().String(),
			ClientSecret: "test-secret",
			Scopes:       "read write",
			Roles:        "admin",
			IsActive:     true,
			CreatedBy:    uuid.New().String(),
		}

		err := ts.StorageProvider.CreateApplication(storageCtx, application)
		require.NoError(t, err)
		assert.NotEmpty(t, application.ID)
		assert.NotZero(t, application.CreatedAt)
		assert.NotZero(t, application.UpdatedAt)
	})

	t.Run("should get application by ID", func(t *testing.T) {
		application := &schemas.Application{
			Name:         "test-m2m-app-by-id-" + uuid.New().String(),
			Description:  "Test M2M application get by ID",
			ClientID:     uuid.New().String(),
			ClientSecret: "test-secret",
			Scopes:       "read",
			Roles:        "user",
			IsActive:     true,
			CreatedBy:    uuid.New().String(),
		}

		err := ts.StorageProvider.CreateApplication(storageCtx, application)
		require.NoError(t, err)
		require.NotEmpty(t, application.ID)

		retrieved, err := ts.StorageProvider.GetApplicationByID(storageCtx, application.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, application.Name, retrieved.Name)
		assert.Equal(t, application.ClientID, retrieved.ClientID)
		assert.Equal(t, application.Description, retrieved.Description)
		assert.Equal(t, application.IsActive, retrieved.IsActive)
	})

	t.Run("should fail to get application with non-existent ID", func(t *testing.T) {
		retrieved, err := ts.StorageProvider.GetApplicationByID(storageCtx, uuid.New().String())
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("should get application by client ID", func(t *testing.T) {
		clientID := uuid.New().String()
		application := &schemas.Application{
			Name:         "test-m2m-app-by-clientid-" + uuid.New().String(),
			Description:  "Test M2M application get by client ID",
			ClientID:     clientID,
			ClientSecret: "test-secret",
			Scopes:       "read write",
			Roles:        "user",
			IsActive:     true,
			CreatedBy:    uuid.New().String(),
		}

		err := ts.StorageProvider.CreateApplication(storageCtx, application)
		require.NoError(t, err)

		retrieved, err := ts.StorageProvider.GetApplicationByClientID(storageCtx, clientID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, clientID, retrieved.ClientID)
		assert.Equal(t, application.Name, retrieved.Name)
	})

	t.Run("should fail to get application with non-existent client ID", func(t *testing.T) {
		retrieved, err := ts.StorageProvider.GetApplicationByClientID(storageCtx, uuid.New().String())
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("should list applications with pagination", func(t *testing.T) {
		// Create two applications to ensure list returns results
		for i := 0; i < 2; i++ {
			app := &schemas.Application{
				Name:         "test-m2m-app-list-" + uuid.New().String(),
				Description:  "Test M2M application for list",
				ClientID:     uuid.New().String(),
				ClientSecret: "test-secret",
				Scopes:       "read",
				Roles:        "user",
				IsActive:     true,
				CreatedBy:    uuid.New().String(),
			}
			err := ts.StorageProvider.CreateApplication(storageCtx, app)
			require.NoError(t, err)
		}

		pagination := &model.Pagination{
			Limit:  10,
			Offset: 0,
		}
		applications, paginationResult, err := ts.StorageProvider.ListApplications(storageCtx, pagination)
		require.NoError(t, err)
		assert.NotNil(t, paginationResult)
		assert.GreaterOrEqual(t, len(applications), 2)
		assert.GreaterOrEqual(t, paginationResult.Total, int64(2))
	})

	t.Run("should update application", func(t *testing.T) {
		application := &schemas.Application{
			Name:         "test-m2m-app-update-" + uuid.New().String(),
			Description:  "Test M2M application before update",
			ClientID:     uuid.New().String(),
			ClientSecret: "test-secret",
			Scopes:       "read",
			Roles:        "user",
			IsActive:     true,
			CreatedBy:    uuid.New().String(),
		}

		err := ts.StorageProvider.CreateApplication(storageCtx, application)
		require.NoError(t, err)
		require.NotEmpty(t, application.ID)

		application.Description = "Test M2M application after update"
		application.Scopes = "read write"
		application.IsActive = false

		err = ts.StorageProvider.UpdateApplication(storageCtx, application)
		require.NoError(t, err)

		retrieved, err := ts.StorageProvider.GetApplicationByID(storageCtx, application.ID)
		require.NoError(t, err)
		assert.Equal(t, "Test M2M application after update", retrieved.Description)
		assert.Equal(t, "read write", retrieved.Scopes)
		assert.False(t, retrieved.IsActive)
	})

	t.Run("should delete application", func(t *testing.T) {
		application := &schemas.Application{
			Name:         "test-m2m-app-delete-" + uuid.New().String(),
			Description:  "Test M2M application for deletion",
			ClientID:     uuid.New().String(),
			ClientSecret: "test-secret",
			Scopes:       "read",
			Roles:        "user",
			IsActive:     true,
			CreatedBy:    uuid.New().String(),
		}

		err := ts.StorageProvider.CreateApplication(storageCtx, application)
		require.NoError(t, err)
		require.NotEmpty(t, application.ID)

		err = ts.StorageProvider.DeleteApplication(storageCtx, application.ID)
		require.NoError(t, err)

		retrieved, err := ts.StorageProvider.GetApplicationByID(storageCtx, application.ID)
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})
}
