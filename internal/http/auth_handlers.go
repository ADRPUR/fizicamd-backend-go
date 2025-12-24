package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"fizicamd-backend-go/internal/services"

	"github.com/google/uuid"
)

type RegisterRequest struct {
	Email           string  `json:"email"`
	Password        string  `json:"password"`
	ConfirmPassword *string `json:"confirmPassword"`
	FirstName       *string `json:"firstName"`
	LastName        *string `json:"lastName"`
	Phone           *string `json:"phone"`
	BirthDate       *string `json:"birthDate"`
	School          *string `json:"school"`
	GradeLevel      *string `json:"gradeLevel"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
	ExpiresAt    int64    `json:"expiresAt"`
	User         *UserDTO `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	if req.ConfirmPassword != nil && req.Password != *req.ConfirmPassword {
		WriteError(w, http.StatusBadRequest, "Password confirmation does not match")
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
	now := time.Now().UTC()
	_, err = s.DB.Exec(`
INSERT INTO users (id, email, password_hash, status, is_email_verified, created_at, updated_at)
VALUES ($1,$2,$3,'ACTIVE',FALSE,$4,$4)
`, userID, email, hash, now)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	_, _ = s.DB.Exec(`
INSERT INTO user_profiles (user_id, first_name, last_name, phone, school, grade_level, birth_date, created_at, updated_at, contact_json, metadata)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8,'{}','{}')
`, userID, req.FirstName, req.LastName, req.Phone, req.School, req.GradeLevel, parseBirthDate(req.BirthDate), now)

	var roleID string
	_ = s.DB.Get(&roleID, `SELECT id FROM roles WHERE code = 'STUDENT'`)
	if roleID != "" {
		_, _ = s.DB.Exec(`INSERT INTO user_roles (id, user_id, role_id, assigned_at) VALUES ($1,$2,$3,$4)`, uuid.NewString(), userID, roleID, now)
		_ = services.EnsureMembership(s.DB, userID, "STUDENT")
	}
	WriteJSON(w, http.StatusOK, map[string]string{"userId": userID, "email": email})
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || strings.TrimSpace(req.Password) == "" {
		WriteError(w, http.StatusBadRequest, "Authentication failed")
		return
	}
	row := struct {
		ID           string `db:"id"`
		PasswordHash string `db:"password_hash"`
		Status       string `db:"status"`
	}{}
	if err := s.DB.Get(&row, `SELECT id, password_hash, status FROM users WHERE lower(email) = $1`, email); err != nil {
		WriteError(w, http.StatusUnauthorized, "Authentication failed")
		return
	}
	if row.Status != "ACTIVE" {
		WriteError(w, http.StatusForbidden, "Authentication failed")
		return
	}
	if !s.Tokens.VerifyPassword(req.Password, row.PasswordHash) {
		WriteError(w, http.StatusUnauthorized, "Authentication failed")
		return
	}
	roles := []string{}
	_ = s.DB.Select(&roles, `
SELECT r.code FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = $1
ORDER BY r.code
`, row.ID)
	access, exp, err := s.Tokens.CreateAccessToken(row.ID, email, roles)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	refresh, err := s.Tokens.CreateRefreshToken(row.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	_ = services.SetLastLogin(s.DB, row.ID)
	userDTO, err := buildUserDTO(s.DB, row.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	WriteJSON(w, http.StatusOK, TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    exp,
		User:         userDTO,
	})
}

func (s *Server) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Authentication failed")
		return
	}
	token, claims, err := s.Tokens.ParseToken(req.RefreshToken)
	if err != nil || !token.Valid || claims["typ"] != "refresh" {
		WriteError(w, http.StatusUnauthorized, "Authentication failed")
		return
	}
	userID, _ := claims["sub"].(string)
	if userID == "" {
		WriteError(w, http.StatusUnauthorized, "Authentication failed")
		return
	}
	roles := []string{}
	_ = s.DB.Select(&roles, `
SELECT r.code FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = $1
ORDER BY r.code
`, userID)
	access, exp, err := s.Tokens.CreateAccessToken(userID, "", roles)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	refresh, err := s.Tokens.CreateRefreshToken(userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	userDTO, err := buildUserDTO(s.DB, userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	WriteJSON(w, http.StatusOK, TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    exp,
		User:         userDTO,
	})
}

func (s *Server) Logout(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func parseBirthDate(raw *string) *time.Time {
	if raw == nil {
		return nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil
	}
	return &parsed
}
