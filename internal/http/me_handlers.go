package httpapi

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"fizicamd-backend-go/internal/services"

	"github.com/go-chi/chi/v5"
)

type ProfileUpdateRequest struct {
	FirstName  *string `json:"firstName"`
	LastName   *string `json:"lastName"`
	BirthDate  *string `json:"birthDate"`
	Gender     *string `json:"gender"`
	Phone      *string `json:"phone"`
	School     *string `json:"school"`
	GradeLevel *string `json:"gradeLevel"`
	Bio        *string `json:"bio"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
}

func (s *Server) Me(w http.ResponseWriter, r *http.Request) {
	userID := CurrentUserID(r)
	userDTO, err := buildUserDTO(s.DB, userID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "User not found")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]*UserDTO{"user": userDTO})
}

func (s *Server) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := CurrentUserID(r)
	var req ProfileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	now := time.Now().UTC()
	_, _ = s.DB.Exec(`
INSERT INTO user_profiles (user_id, created_at, updated_at, contact_json, metadata)
VALUES ($1,$2,$2,'{}','{}')
ON CONFLICT (user_id) DO NOTHING
`, userID, now)
	_, err := s.DB.Exec(`
UPDATE user_profiles
SET first_name = $2,
    last_name = $3,
    birth_date = $4,
    gender = $5,
    phone = $6,
    school = $7,
    grade_level = $8,
    bio = $9,
    updated_at = $10
WHERE user_id = $1
`, userID, req.FirstName, req.LastName, parseBirthDate(req.BirthDate), req.Gender, req.Phone, req.School, req.GradeLevel, req.Bio, now)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	userDTO, err := buildUserDTO(s.DB, userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]*UserDTO{"user": userDTO})
}

func (s *Server) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := CurrentUserID(r)
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	if req.NewPassword != req.ConfirmPassword {
		WriteError(w, http.StatusBadRequest, "Password confirmation does not match")
		return
	}
	row := struct {
		PasswordHash string `db:"password_hash"`
	}{}
	if err := s.DB.Get(&row, `SELECT password_hash FROM users WHERE id = $1`, userID); err != nil {
		WriteError(w, http.StatusNotFound, "User not found")
		return
	}
	if !s.Tokens.VerifyPassword(req.CurrentPassword, row.PasswordHash) {
		WriteError(w, http.StatusUnauthorized, "Authentication failed")
		return
	}
	hash, err := s.Tokens.HashPassword(req.NewPassword)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	_, err = s.DB.Exec(`UPDATE users SET password_hash = $1 WHERE id = $2`, hash, userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID := CurrentUserID(r)
	var avatarID *string
	_ = s.DB.Get(&avatarID, `SELECT avatar_media_id FROM user_profiles WHERE user_id = $1`, userID)
	if avatarID != nil && strings.TrimSpace(*avatarID) != "" {
		_ = services.DeleteAsset(s.DB, s.Config.MediaStoragePath, *avatarID)
	}
	_, _ = s.DB.Exec(`DELETE FROM users WHERE id = $1`, userID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) Ping(w http.ResponseWriter, r *http.Request) {
	userID := CurrentUserID(r)
	_ = services.TouchLastSeen(s.DB, userID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID := CurrentUserID(r)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		WriteError(w, http.StatusBadRequest, "Fișierul este gol.")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Fișierul este gol.")
		return
	}
	defer file.Close()
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	assetID, url, err := services.SaveMediaAsset(s.DB, s.Config.MediaStoragePath, services.BucketUsers, contentType, header.Filename, "AVATAR", userID, file)
	if err != nil {
		if serr, ok := err.(services.ServiceError); ok {
			WriteError(w, serr.Status, serr.Message)
			return
		}
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	var previous *string
	_ = s.DB.Get(&previous, `SELECT avatar_media_id FROM user_profiles WHERE user_id = $1`, userID)
	_, _ = s.DB.Exec(`
INSERT INTO user_profiles (user_id, created_at, updated_at, contact_json, metadata)
VALUES ($1,$2,$2,'{}','{}')
ON CONFLICT (user_id) DO NOTHING
`, userID, time.Now().UTC())
	_, _ = s.DB.Exec(`UPDATE user_profiles SET avatar_media_id = $1, updated_at = $2 WHERE user_id = $3`, assetID, time.Now().UTC(), userID)
	if previous != nil && *previous != "" && *previous != assetID {
		_ = services.DeleteAsset(s.DB, s.Config.MediaStoragePath, *previous)
	}
	WriteJSON(w, http.StatusOK, map[string]string{"assetId": assetID, "url": url})
}

func (s *Server) UploadResource(w http.ResponseWriter, r *http.Request) {
	userID := CurrentUserID(r)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		WriteError(w, http.StatusBadRequest, "Fișierul este gol.")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Fișierul este gol.")
		return
	}
	defer file.Close()
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	mediaType := "OTHER"
	if strings.HasPrefix(contentType, "image/") {
		mediaType = "IMAGE"
	} else if strings.EqualFold(contentType, "application/pdf") {
		mediaType = "DOCUMENT"
	}
	assetID, url, err := services.SaveMediaAsset(s.DB, s.Config.MediaStoragePath, services.BucketResources, contentType, header.Filename, mediaType, userID, file)
	if err != nil {
		if serr, ok := err.(services.ServiceError); ok {
			WriteError(w, serr.Status, serr.Message)
			return
		}
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"assetId": assetID, "url": url})
}

func (s *Server) MediaContent(w http.ResponseWriter, r *http.Request) {
	assetID := chi.URLParam(r, "assetId")
	row := struct {
		Bucket      string  `db:"bucket"`
		StorageKey  string  `db:"storage_key"`
		Filename    *string `db:"filename"`
		ContentType string  `db:"content_type"`
	}{}
	if err := s.DB.Get(&row, `SELECT bucket, storage_key, filename, content_type FROM media_assets WHERE id = $1`, assetID); err != nil {
		WriteError(w, http.StatusNotFound, "Media negăsită")
		return
	}
	path := filepath.Join(s.Config.MediaStoragePath, row.Bucket, row.StorageKey)
	if row.Filename != nil {
		w.Header().Set("Content-Disposition", "inline; filename=\""+*row.Filename+"\"")
	}
	if row.ContentType != "" {
		w.Header().Set("Content-Type", row.ContentType)
	}
	http.ServeFile(w, r, path)
}
