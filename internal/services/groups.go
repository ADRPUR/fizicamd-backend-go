package services

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var roleCodes = []string{"ADMIN", "TEACHER", "STUDENT"}

func EnsureRoleGroups(db *sqlx.DB) error {
	for _, code := range roleCodes {
		if err := ensureRoleGroup(db, code); err != nil {
			return err
		}
	}
	return nil
}

func ensureRoleGroup(db *sqlx.DB, code string) error {
	name := "Role: " + code
	var exists bool
	if err := db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM groups WHERE name = $1)`, name); err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err := db.Exec(`
INSERT INTO groups (id, name, description, visibility, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$5)
`, uuid.NewString(), name, "System generated group for role "+code, "SYSTEM", time.Now().UTC())
	return err
}

func EnsureUserMemberships(db *sqlx.DB, userID string) error {
	roles := []string{}
	if err := db.Select(&roles, `
SELECT r.code
FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = $1
`, userID); err != nil {
		return err
	}
	for _, code := range roles {
		if err := ensureMembership(db, userID, code); err != nil {
			return err
		}
	}
	return nil
}

func ensureMembership(db *sqlx.DB, userID, roleCode string) error {
	roleCode = strings.ToUpper(roleCode)
	name := "Role: " + roleCode
	var groupID string
	if err := db.Get(&groupID, `SELECT id FROM groups WHERE name = $1`, name); err != nil {
		return err
	}
	var exists bool
	if err := db.Get(&exists, `
SELECT EXISTS(
  SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2
)
`, groupID, userID); err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err := db.Exec(`
INSERT INTO group_members (id, group_id, user_id, member_role, status, joined_at, created_at, updated_at)
VALUES ($1,$2,$3,$4,'ACTIVE',$5,$5,$5)
`, uuid.NewString(), groupID, userID, roleCode, time.Now().UTC())
	return err
}

func EnsureMembership(db *sqlx.DB, userID, roleCode string) error {
	return ensureMembership(db, userID, roleCode)
}

func CreateGroup(db *sqlx.DB, name string, grade *int, year *int) (string, error) {
	id := uuid.NewString()
	_, err := db.Exec(`
INSERT INTO groups (id, name, grade, year, visibility, created_at, updated_at)
VALUES ($1,$2,$3,$4,'PRIVATE',$5,$5)
`, id, strings.TrimSpace(name), grade, year, time.Now().UTC())
	return id, err
}

func UpdateGroup(db *sqlx.DB, groupID string, name *string, grade *int, year *int) error {
	_, err := db.Exec(`
UPDATE groups
SET name = COALESCE($2, name), grade = $3, year = $4, updated_at = $5
WHERE id = $1
`, groupID, name, grade, year, time.Now().UTC())
	return err
}

func DeleteGroup(db *sqlx.DB, groupID string) error {
	_, err := db.Exec(`DELETE FROM groups WHERE id = $1`, groupID)
	return err
}

func AddMember(db *sqlx.DB, groupID, userID, role string) error {
	role = strings.ToUpper(role)
	_, err := db.Exec(`
INSERT INTO group_members (id, group_id, user_id, member_role, status, joined_at, created_at, updated_at)
VALUES ($1,$2,$3,$4,'ACTIVE',$5,$5,$5)
ON CONFLICT (group_id, user_id) DO UPDATE SET member_role = EXCLUDED.member_role
`, uuid.NewString(), groupID, userID, role, time.Now().UTC())
	return err
}

func RemoveMember(db *sqlx.DB, groupID, userID string) error {
	_, err := db.Exec(`DELETE FROM group_members WHERE group_id = $1 AND user_id = $2`, groupID, userID)
	return err
}
