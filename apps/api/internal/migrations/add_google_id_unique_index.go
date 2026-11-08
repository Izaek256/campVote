package migrations

import "gorm.io/gorm"

// AddGoogleIDUniqueIndex makes one Google subject belong to at most one
// CampVote account.
//
// The users table has carried google_id since the redirect OAuth provider was
// added, but nothing stopped two rows sharing a subject. A partial unique
// index does: empty values are the "not linked" state and are excluded, so
// existing password accounts and every new account are unaffected.
//
// Duplicate non-empty values are cleared on all but the oldest row first. That
// is not a merge: the accounts stay, and the cleared account links again the
// next time its owner signs in. Without this step the index would refuse to be
// created on a database that already held a duplicate.
//
// Postgres (including Neon) and SQLite both support partial indexes. Other
// dialects are left alone rather than given an index that would reject every
// second account with an empty google_id.
func AddGoogleIDUniqueIndex(db *gorm.DB) error {
	switch db.Dialector.Name() {
	case "postgres", "sqlite":
	default:
		return nil
	}

	if err := db.Exec(`
		UPDATE users
		SET google_id = ''
		WHERE google_id <> ''
		  AND id NOT IN (
			SELECT id FROM (
				SELECT id, ROW_NUMBER() OVER (PARTITION BY google_id ORDER BY created_at, id) AS rn
				FROM users
				WHERE google_id <> ''
			) ranked
			WHERE rn = 1
		  )
	`).Error; err != nil {
		return err
	}

	return db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_users_google_id
		ON users (google_id)
		WHERE google_id <> ''
	`).Error
}
