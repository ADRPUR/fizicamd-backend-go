package services

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

func FetchRoles(db *sqlx.DB, userID string) ([]string, error) {
	roles := []string{}
	err := db.Select(&roles, `
SELECT r.code
FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = $1
ORDER BY r.code
`, userID)
	return roles, err
}

func HasRole(db *sqlx.DB, userID, role string) (bool, error) {
	var exists bool
	err := db.Get(&exists, `
SELECT EXISTS(
  SELECT 1
  FROM roles r
  JOIN user_roles ur ON ur.role_id = r.id
  WHERE ur.user_id = $1 AND r.code = $2
)
`, userID, role)
	return exists, err
}

func TouchLastSeen(db *sqlx.DB, userID string) error {
	_, err := db.Exec(`UPDATE users SET last_seen_at = $1 WHERE id = $2`, time.Now().UTC(), userID)
	return err
}

func SetLastLogin(db *sqlx.DB, userID string) error {
	now := time.Now().UTC()
	_, err := db.Exec(`UPDATE users SET last_login_at = $1, last_seen_at = $1 WHERE id = $2`, now, userID)
	return err
}

func GetUserStatus(db *sqlx.DB, userID string) (string, error) {
	var status sql.NullString
	err := db.Get(&status, `SELECT status FROM users WHERE id = $1`, userID)
	if err != nil {
		return "", err
	}
	if status.Valid {
		return status.String, nil
	}
	return "", nil
}
