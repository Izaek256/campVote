package migrations

import "gorm.io/gorm"

// AddLandingIndexes adds the partial index behind the public landing page's
// upcoming-elections query:
//
//	WHERE public_listing AND status = 'DRAFT' AND starts_at > now()
//
// The predicate keeps the index small: only the handful of elections an
// administrator has explicitly published to the landing page are in it.
func AddLandingIndexes(db *gorm.DB) error {
	return db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_elections_public_upcoming
		ON elections (starts_at)
		WHERE public_listing AND status = 'DRAFT'
	`).Error
}
