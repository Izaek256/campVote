package database_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"campvote/apps/api/internal/database"
	"campvote/apps/api/internal/models"
)

func newBootstrapDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}))
	return db
}

func TestCreateFirstAdminCreatesAVerifiedAdministrator(t *testing.T) {
	db := newBootstrapDB(t)

	admin, err := database.CreateFirstAdmin(context.Background(), db,
		"First.Admin@must.ac.ug", "harbour-lamp-97", "First", "Admin")
	require.NoError(t, err)

	assert.Equal(t, models.RoleAdmin, admin.Role)
	assert.True(t, admin.Active)
	assert.NotNil(t, admin.EmailVerifiedAt)
	assert.Equal(t, "first.admin@must.ac.ug", admin.Email)
	assert.NotEqual(t, "harbour-lamp-97", admin.Password, "the password must be stored as a hash")
	assert.True(t, admin.CheckPassword("harbour-lamp-97"))

	// One-time: a second run must not plant another administrator.
	_, err = database.CreateFirstAdmin(context.Background(), db,
		"second.admin@must.ac.ug", "another-lamp-97", "", "")
	assert.ErrorIs(t, err, database.ErrAdministratorExists)
}

func TestCreateFirstAdminRejectsWeakOrMissingCredentials(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
	}{
		{"no email", "", "harbour-lamp-97"},
		{"no password", "admin@must.ac.ug", ""},
		{"common password", "admin@must.ac.ug", "admin123"},
		{"too short", "admin@must.ac.ug", "short1"},
		{"password from the account", "admin@must.ac.ug", "admin-must-123"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := newBootstrapDB(t)
			_, err := database.CreateFirstAdmin(context.Background(), db, tc.email, tc.password, "First", "Admin")
			require.Error(t, err)

			var count int64
			require.NoError(t, db.Model(&models.User{}).Count(&count).Error)
			assert.Zero(t, count, "a rejected bootstrap must not create an account")
		})
	}
}

func TestCreateFirstAdminDoesNotMatchASoftDeletedAdministrator(t *testing.T) {
	db := newBootstrapDB(t)
	admin, err := database.CreateFirstAdmin(context.Background(), db,
		"admin@must.ac.ug", "harbour-lamp-97", "", "")
	require.NoError(t, err)
	require.NoError(t, db.Delete(admin).Error)

	// A deleted administrator still counts: resurrecting bootstrap because the
	// only admin was removed would hand out a fresh known account.
	_, err = database.CreateFirstAdmin(context.Background(), db,
		"replacement@must.ac.ug", "harbour-lamp-98", "", "")
	assert.True(t, errors.Is(err, database.ErrAdministratorExists))
}
