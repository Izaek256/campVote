// The external test package is deliberate: models imports migrations, so an
// in-package test that imports models would not compile.
package migrations_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"campvote/apps/api/internal/migrations"
	"campvote/apps/api/internal/models"
)

func googleIndexDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&models.User{}))
	return db
}

func TestAddGoogleIDUniqueIndex_DeduplicatesAndEnforcesUniqueness(t *testing.T) {
	db := googleIndexDB(t)
	first := models.User{FirstName: "Ada", LastName: "L", Email: "ada@example.com", Active: true, GoogleID: "same-sub"}
	second := models.User{FirstName: "Grace", LastName: "H", Email: "grace@example.com", Active: true, GoogleID: "same-sub"}
	require.NoError(t, db.Create(&first).Error)
	require.NoError(t, db.Create(&second).Error)

	require.NoError(t, migrations.AddGoogleIDUniqueIndex(db))
	// Running again on every boot must be safe.
	require.NoError(t, migrations.AddGoogleIDUniqueIndex(db))

	var kept, cleared int64
	require.NoError(t, db.Model(&models.User{}).Where("google_id = ?", "same-sub").Count(&kept).Error)
	require.NoError(t, db.Model(&models.User{}).Where("id = ? AND google_id = ?", second.ID, "").Count(&cleared).Error)
	assert.EqualValues(t, 1, kept, "only the oldest row keeps a duplicated subject")
	assert.EqualValues(t, 1, cleared, "the other account remains, unlinked")

	third := models.User{FirstName: "Alan", LastName: "T", Email: "alan@example.com", Active: true, GoogleID: "same-sub"}
	err := db.Create(&third).Error
	assert.Error(t, err, "the partial unique index must refuse a second row with the same subject")
}
