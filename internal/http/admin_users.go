package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fizicamd-backend-go/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AdminUserResponse struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	Status      string     `json:"status"`
	PrimaryRole string     `json:"primaryRole"`
	Roles       []string   `json:"roles"`
	FirstName   *string    `json:"firstName,omitempty"`
	LastName    *string    `json:"lastName,omitempty"`
	Phone       *string    `json:"phone,omitempty"`
	School      *string    `json:"school,omitempty"`
	GradeLevel  *string    `json:"gradeLevel,omitempty"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
	LastSeenAt  *time.Time `json:"lastSeenAt,omitempty"`
}

type PagedResponse struct {
	Items    []AdminUserResponse `json:"items"`
	Total    int                 `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"pageSize"`
}

type AdminUserCreateRequest struct {
	Email      string   `json:"email"`
	Password   string   `json:"password"`
	Roles      []string `json:"roles"`
	FirstName  *string  `json:"firstName"`
	LastName   *string  `json:"lastName"`
	Phone      *string  `json:"phone"`
	School     *string  `json:"school"`
	GradeLevel *string  `json:"gradeLevel"`
	Status     *string  `json:"status"`
}

type AdminUserUpdateRequest struct {
	Email      string   `json:"email"`
	Roles      []string `json:"roles"`
	FirstName  *string  `json:"firstName"`
	LastName   *string  `json:"lastName"`
	Phone      *string  `json:"phone"`
	School     *string  `json:"school"`
	GradeLevel *string  `json:"gradeLevel"`
	Status     *string  `json:"status"`
}

type AssignRoleRequest struct {
	Role string `json:"role"`
}

func (s *Server) ListUsers(w http.ResponseWriter, r *http.Request) {
	page := parseInt(r.URL.Query().Get("page"), 1)
	pageSize := parseInt(r.URL.Query().Get("pageSize"), 10)
	if pageSize > 100 {
		pageSize = 100
	}
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	args := []interface{}{}
	where := ""
	if search != "" {
		where = "WHERE lower(email) LIKE $1"
		args = append(args, "%"+strings.ToLower(search)+"%")
	}
	var total int
	if err := s.DB.Get(&total, "SELECT count(*) FROM users "+where, args...); err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	offset := (page - 1) * pageSize
	query := `
SELECT u.id, u.email, u.status, u.created_at, u.last_login_at, u.last_seen_at,
       p.first_name, p.last_name, p.phone, p.school, p.grade_level
FROM users u
LEFT JOIN user_profiles p ON p.user_id = u.id
` + where + `
ORDER BY u.created_at DESC
LIMIT $%d OFFSET $%d`
	args = append(args, pageSize, offset)
	query = fmt.Sprintf(query, len(args)-1, len(args))
	rows := []struct {
		ID        string     `db:"id"`
		Email     string     `db:"email"`
		Status    string     `db:"status"`
		CreatedAt *time.Time `db:"created_at"`
		LastLogin *time.Time `db:"last_login_at"`
		LastSeen  *time.Time `db:"last_seen_at"`
		FirstName *string    `db:"first_name"`
		LastName  *string    `db:"last_name"`
		Phone     *string    `db:"phone"`
		School    *string    `db:"school"`
		Grade     *string    `db:"grade_level"`
	}{}
	if err := s.DB.Select(&rows, query, args...); err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	items := make([]AdminUserResponse, 0, len(rows))
	for _, row := range rows {
		roles := []string{}
		_ = s.DB.Select(&roles, `SELECT r.code FROM roles r JOIN user_roles ur ON ur.role_id = r.id WHERE ur.user_id = $1`, row.ID)
		primary := "STUDENT"
		if len(roles) > 0 {
			primary = roles[0]
		}
		items = append(items, AdminUserResponse{
			ID:          row.ID,
			Email:       row.Email,
			Status:      row.Status,
			PrimaryRole: primary,
			Roles:       roles,
			FirstName:   row.FirstName,
			LastName:    row.LastName,
			Phone:       row.Phone,
			School:      row.School,
			GradeLevel:  row.Grade,
			CreatedAt:   row.CreatedAt,
			LastLoginAt: row.LastLogin,
			LastSeenAt:  row.LastSeen,
		})
	}
	WriteJSON(w, http.StatusOK, PagedResponse{Items: items, Total: total, Page: page, PageSize: pageSize})
}

func (s *Server) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req AdminUserCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || strings.TrimSpace(req.Password) == "" {
		WriteError(w, http.StatusBadRequest, "Email and password are required")
		return
	}
	var exists bool
	if err := s.DB.Get(&exists, `SELECT EXISTS(SELECT 1 FROM users WHERE lower(email) = $1)`, email); err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	if exists {
		WriteError(w, http.StatusBadRequest, "User already exists")
		return
	}
	hash, err := s.Tokens.HashPassword(req.Password)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	userID := uuid.NewString()
	status := "ACTIVE"
	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		status = strings.ToUpper(strings.TrimSpace(*req.Status))
	}
	now := time.Now().UTC()
	_, err = s.DB.Exec(`
INSERT INTO users (id, email, password_hash, status, is_email_verified, created_at, updated_at)
VALUES ($1,$2,$3,$4,false,$5,$5)
`, userID, email, hash, status, now)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	_, _ = s.DB.Exec(`
INSERT INTO user_profiles (user_id, first_name, last_name, phone, school, grade_level, created_at, updated_at, contact_json, metadata)
VALUES ($1,$2,$3,$4,$5,$6,$7,$7,'{}','{}')
`, userID, req.FirstName, req.LastName, req.Phone, req.School, req.GradeLevel, now)
	roles := req.Roles
	if len(roles) == 0 {
		roles = []string{"STUDENT"}
	}
	for _, role := range roles {
		role = strings.ToUpper(strings.TrimSpace(role))
		var roleID string
		if err := s.DB.Get(&roleID, `SELECT id FROM roles WHERE code = $1`, role); err == nil && roleID != "" {
			_, _ = s.DB.Exec(`INSERT INTO user_roles (id, user_id, role_id, assigned_at) VALUES ($1,$2,$3,$4)`, uuid.NewString(), userID, roleID, now)
			_ = services.EnsureMembership(s.DB, userID, role)
		}
	}
	resp, err := s.buildAdminUser(userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (s *Server) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	var req AdminUserUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		WriteError(w, http.StatusBadRequest, "Email is required")
		return
	}
	var existing string
	if err := s.DB.Get(&existing, `SELECT email FROM users WHERE id = $1`, userID); err != nil {
		WriteError(w, http.StatusNotFound, "User not found")
		return
	}
	if strings.ToLower(existing) != email {
		WriteError(w, http.StatusBadRequest, "Email-ul nu poate fi modificat.")
		return
	}
	status := (*string)(nil)
	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		value := strings.ToUpper(strings.TrimSpace(*req.Status))
		status = &value
	}
	_, _ = s.DB.Exec(`UPDATE users SET status = COALESCE($2, status), updated_at = $3 WHERE id = $1`, userID, status, time.Now().UTC())
	_, _ = s.DB.Exec(`
INSERT INTO user_profiles (user_id, created_at, updated_at, contact_json, metadata)
VALUES ($1,$2,$2,'{}','{}')
ON CONFLICT (user_id) DO NOTHING
`, userID, time.Now().UTC())
	_, _ = s.DB.Exec(`
UPDATE user_profiles
SET first_name = $2, last_name = $3, phone = $4, school = $5, grade_level = $6, updated_at = $7
WHERE user_id = $1
`, userID, req.FirstName, req.LastName, req.Phone, req.School, req.GradeLevel, time.Now().UTC())

	roles := req.Roles
	if len(roles) == 0 {
		roles = []string{"STUDENT"}
	}
	current := []string{}
	_ = s.DB.Select(&current, `SELECT r.code FROM roles r JOIN user_roles ur ON ur.role_id = r.id WHERE ur.user_id = $1`, userID)
	currentSet := map[string]bool{}
	for _, role := range current {
		currentSet[role] = true
	}
	desiredSet := map[string]bool{}
	for _, role := range roles {
		role = strings.ToUpper(strings.TrimSpace(role))
		if role != "" {
			desiredSet[role] = true
		}
	}
	for role := range desiredSet {
		if currentSet[role] {
			continue
		}
		var roleID string
		if err := s.DB.Get(&roleID, `SELECT id FROM roles WHERE code = $1`, role); err == nil && roleID != "" {
			_, _ = s.DB.Exec(`INSERT INTO user_roles (id, user_id, role_id, assigned_at) VALUES ($1,$2,$3,$4)`, uuid.NewString(), userID, roleID, time.Now().UTC())
			_ = services.EnsureMembership(s.DB, userID, role)
		}
	}
	for role := range currentSet {
		if desiredSet[role] {
			continue
		}
		var roleID string
		if err := s.DB.Get(&roleID, `SELECT id FROM roles WHERE code = $1`, role); err == nil && roleID != "" {
			_, _ = s.DB.Exec(`DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`, userID, roleID)
		}
	}
	resp, err := s.buildAdminUser(userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (s *Server) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	var avatarID *string
	_ = s.DB.Get(&avatarID, `SELECT avatar_media_id FROM user_profiles WHERE user_id = $1`, userID)
	if avatarID != nil && *avatarID != "" {
		_ = services.DeleteAsset(s.DB, s.Config.MediaStoragePath, *avatarID)
	}
	_, _ = s.DB.Exec(`DELETE FROM users WHERE id = $1`, userID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) AssignRole(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	var req AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	role := strings.ToUpper(strings.TrimSpace(req.Role))
	if role == "" {
		WriteError(w, http.StatusBadRequest, "Role not found")
		return
	}
	var roleID string
	if err := s.DB.Get(&roleID, `SELECT id FROM roles WHERE code = $1`, role); err != nil {
		WriteError(w, http.StatusNotFound, "Role not found")
		return
	}
	_, _ = s.DB.Exec(`INSERT INTO user_roles (id, user_id, role_id, assigned_at) VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`, uuid.NewString(), userID, roleID, time.Now().UTC())
	_ = services.EnsureMembership(s.DB, userID, role)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) RemoveRole(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	role := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "role")))
	var roleID string
	if err := s.DB.Get(&roleID, `SELECT id FROM roles WHERE code = $1`, role); err != nil {
		WriteError(w, http.StatusNotFound, "Role not found")
		return
	}
	_, _ = s.DB.Exec(`DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`, userID, roleID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) buildAdminUser(userID string) (AdminUserResponse, error) {
	row := struct {
		ID        string     `db:"id"`
		Email     string     `db:"email"`
		Status    string     `db:"status"`
		CreatedAt *time.Time `db:"created_at"`
		LastLogin *time.Time `db:"last_login_at"`
		LastSeen  *time.Time `db:"last_seen_at"`
		FirstName *string    `db:"first_name"`
		LastName  *string    `db:"last_name"`
		Phone     *string    `db:"phone"`
		School    *string    `db:"school"`
		Grade     *string    `db:"grade_level"`
	}{}
	if err := s.DB.Get(&row, `
SELECT u.id, u.email, u.status, u.created_at, u.last_login_at, u.last_seen_at,
       p.first_name, p.last_name, p.phone, p.school, p.grade_level
FROM users u
LEFT JOIN user_profiles p ON p.user_id = u.id
WHERE u.id = $1
`, userID); err != nil {
		return AdminUserResponse{}, err
	}
	roles := []string{}
	_ = s.DB.Select(&roles, `SELECT r.code FROM roles r JOIN user_roles ur ON ur.role_id = r.id WHERE ur.user_id = $1`, userID)
	primary := "STUDENT"
	if len(roles) > 0 {
		primary = roles[0]
	}
	return AdminUserResponse{
		ID:          row.ID,
		Email:       row.Email,
		Status:      row.Status,
		PrimaryRole: primary,
		Roles:       roles,
		FirstName:   row.FirstName,
		LastName:    row.LastName,
		Phone:       row.Phone,
		School:      row.School,
		GradeLevel:  row.Grade,
		CreatedAt:   row.CreatedAt,
		LastLoginAt: row.LastLogin,
		LastSeenAt:  row.LastSeen,
	}, nil
}

func parseInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if value < 1 {
		return fallback
	}
	return value
}
