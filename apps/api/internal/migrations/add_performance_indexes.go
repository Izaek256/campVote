package migrations

import (
	"gorm.io/gorm"
)

// AddPerformanceIndexes adds composite indexes for common query patterns
func AddPerformanceIndexes(db *gorm.DB) error {
	// Composite index for querying open elections by status and time
	// Optimizes: WHERE status='OPEN' AND starts_at <= NOW() AND ends_at > NOW()
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_elections_status_schedule 
		ON elections (status, starts_at, ends_at)
		WHERE status IN ('OPEN', 'CLOSED')
	`).Error; err != nil {
		return err
	}

	// Composite index for candidate queries by election and status
	// Optimizes: WHERE election_id=? AND approval_status=?
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_candidates_election_status 
		ON candidates (election_id, approval_status)
	`).Error; err != nil {
		return err
	}

	// Composite index for ballot queries by election and user
	// Optimizes: WHERE election_id=? AND user_id=?
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_ballots_election_user 
		ON ballots (election_id, user_id)
	`).Error; err != nil {
		return err
	}

	// Index for users by role and active status
	// Optimizes: WHERE role=? AND active=?
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_users_role_active 
		ON users (role, active)
	`).Error; err != nil {
		return err
	}

	return nil
}
