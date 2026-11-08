package migrations

import (
	"gorm.io/gorm"
)

// AddCandidateUserID adds the user_id column to the candidates table
// This column links a candidate to their user account
func AddCandidateUserID(db *gorm.DB) error {
	// Check if column exists first to avoid errors on rerun
	switch db.Dialector.Name() {
	case "postgres":
		// PostgreSQL supports IF NOT EXISTS
		return db.Exec(`
			ALTER TABLE candidates
			ADD COLUMN IF NOT EXISTS user_id VARCHAR(36);
		`).Error
	case "mysql":
		// MySQL doesn't have IF NOT EXISTS for columns, so we check first
		var columnExists int
		db.Raw(`
			SELECT COUNT(*)
			FROM information_schema.columns
			WHERE table_schema = DATABASE()
			AND table_name = 'candidates'
			AND column_name = 'user_id'
		`).Scan(&columnExists)
		if columnExists > 0 {
			return nil
		}
		return db.Exec(`
			ALTER TABLE candidates
			ADD COLUMN user_id VARCHAR(36);
		`).Error
	case "sqlite":
		// SQLite doesn't have IF NOT EXISTS for columns, so we check first
		var columnExists int
		db.Raw(`
			SELECT COUNT(*)
			FROM pragma_table_info('candidates')
			WHERE name = 'user_id'
		`).Scan(&columnExists)
		if columnExists > 0 {
			return nil
		}
		return db.Exec(`
			ALTER TABLE candidates
			ADD COLUMN user_id TEXT;
		`).Error
	default:
		// For other databases, try with IF NOT EXISTS syntax
		return db.Exec(`
			ALTER TABLE candidates
			ADD COLUMN IF NOT EXISTS user_id VARCHAR(36);
		`).Error
	}
}
