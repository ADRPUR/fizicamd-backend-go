package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"fizicamd-backend-go/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ResourceCreateRequest struct {
	CategoryCode string          `json:"categoryCode"`
	Title        string          `json:"title"`
	Summary      string          `json:"summary"`
	AvatarID     *string         `json:"avatarAssetId"`
	Tags         []string        `json:"tags"`
	Blocks       json.RawMessage `json:"blocks"`
	Status       string          `json:"status"`
}

func (s *Server) TeacherListResources(w http.ResponseWriter, r *http.Request) {
	userID := CurrentUserID(r)
	canManage := hasRole(CurrentRoles(r), "ADMIN")
	rows := []struct {
		ID        string     `db:"id"`
		Category  string     `db:"category_code"`
		AuthorID  string     `db:"author_id"`
		Title     string     `db:"title"`
		Slug      string     `db:"slug"`
		Summary   string     `db:"summary"`
		AvatarID  *string    `db:"avatar_media_id"`
		Tags      []byte     `db:"tags"`
		Status    string     `db:"status"`
		Published *time.Time `db:"published_at"`
	}{}
	query := `
SELECT id, category_code, author_id, title, slug, summary, avatar_media_id, tags, status, published_at
FROM resource_entries
`
	args := []interface{}{}
	if !canManage {
		query += "WHERE author_id = $1\n"
		args = append(args, userID)
	}
	query += "ORDER BY created_at DESC"
	if err := s.DB.Select(&rows, query, args...); err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	items := make([]ResourceCardDTO, 0, len(rows))
	for _, row := range rows {
		categoryDTO := s.fetchCategory(row.Category)
		tags := []string{}
		_ = json.Unmarshal(row.Tags, &tags)
		var avatarURL *string
		if row.AvatarID != nil {
			url := services.BuildAssetURL(*row.AvatarID)
			avatarURL = &url
		}
		var published *string
		if row.Published != nil {
			formatted := row.Published.UTC().Format(time.RFC3339)
			published = &formatted
		}
		author := s.authorDisplayName(row.AuthorID)
		items = append(items, ResourceCardDTO{
			ID:          row.ID,
			Title:       row.Title,
			Slug:        row.Slug,
			Summary:     row.Summary,
			Category:    categoryDTO,
			AvatarURL:   avatarURL,
			Tags:        tags,
			AuthorName:  author,
			PublishedAt: published,
			Status:      row.Status,
		})
	}
	WriteJSON(w, http.StatusOK, map[string][]ResourceCardDTO{"items": items})
}

func (s *Server) TeacherResourceDetail(w http.ResponseWriter, r *http.Request) {
	resourceID := chi.URLParam(r, "resourceId")
	row := struct {
		ID        string     `db:"id"`
		Category  string     `db:"category_code"`
		AuthorID  string     `db:"author_id"`
		Title     string     `db:"title"`
		Slug      string     `db:"slug"`
		Summary   string     `db:"summary"`
		AvatarID  *string    `db:"avatar_media_id"`
		Tags      []byte     `db:"tags"`
		Status    string     `db:"status"`
		Published *time.Time `db:"published_at"`
		Content   []byte     `db:"content"`
	}{}
	if err := s.DB.Get(&row, `
SELECT id, category_code, author_id, title, slug, summary, avatar_media_id, tags, status, published_at, content
FROM resource_entries
WHERE id = $1
`, resourceID); err != nil {
		WriteError(w, http.StatusNotFound, "Resource not found")
		return
	}
	categoryDTO := s.fetchCategory(row.Category)
	tags := []string{}
	_ = json.Unmarshal(row.Tags, &tags)
	var avatarURL *string
	if row.AvatarID != nil {
		url := services.BuildAssetURL(*row.AvatarID)
		avatarURL = &url
	}
	var published *string
	if row.Published != nil {
		formatted := row.Published.UTC().Format(time.RFC3339)
		published = &formatted
	}
	author := s.authorDisplayName(row.AuthorID)
	WriteJSON(w, http.StatusOK, ResourceDetailDTO{
		ID:            row.ID,
		Title:         row.Title,
		Slug:          row.Slug,
		Summary:       row.Summary,
		Category:      categoryDTO,
		AvatarURL:     avatarURL,
		AvatarAssetID: row.AvatarID,
		Tags:          tags,
		AuthorName:    author,
		PublishedAt:   published,
		Status:        row.Status,
		Blocks:        json.RawMessage(row.Content),
	})
}

func (s *Server) CreateResource(w http.ResponseWriter, r *http.Request) {
	userID := CurrentUserID(r)
	var req ResourceCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	categoryCode := strings.TrimSpace(req.CategoryCode)
	title := strings.TrimSpace(req.Title)
	summary := strings.TrimSpace(req.Summary)
	if categoryCode == "" || title == "" || summary == "" {
		WriteError(w, http.StatusBadRequest, "Titlul și descrierea sunt obligatorii.")
		return
	}
	var categoryExists bool
	_ = s.DB.Get(&categoryExists, `SELECT EXISTS(SELECT 1 FROM resource_categories WHERE code = $1)`, categoryCode)
	if !categoryExists {
		WriteError(w, http.StatusBadRequest, "Categoria selectată nu există.")
		return
	}
	blocks, err := services.ValidateBlocks(req.Blocks)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	blockJSON, _ := json.Marshal(blocks)
	tags := services.CleanTags(req.Tags)
	tagsJSON, _ := json.Marshal(tags)
	status := req.Status
	if status == "" {
		status = "PUBLISHED"
	}
	now := time.Now().UTC()
	resourceID := uuid.NewString()
	slug, err := services.ResolveResourceSlug(s.DB, title)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	publishedAt := (*time.Time)(nil)
	if status == "PUBLISHED" {
		publishedAt = &now
	}
	_, err = s.DB.Exec(`
INSERT INTO resource_entries (id, category_code, author_id, title, slug, summary, avatar_media_id, tags, content, status, published_at, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$12)
`, resourceID, categoryCode, userID, title, slug, summary, req.AvatarID, tagsJSON, blockJSON, status, publishedAt, now)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	row := struct {
		Published *time.Time `db:"published_at"`
	}{}
	_ = s.DB.Get(&row, `SELECT published_at FROM resource_entries WHERE id = $1`, resourceID)
	var published *string
	if row.Published != nil {
		formatted := row.Published.UTC().Format(time.RFC3339)
		published = &formatted
	}
	categoryDTO := s.fetchCategory(categoryCode)
	author := s.authorDisplayName(userID)
	var avatarURL *string
	if req.AvatarID != nil {
		url := services.BuildAssetURL(*req.AvatarID)
		avatarURL = &url
	}
	WriteJSON(w, http.StatusOK, ResourceDetailDTO{
		ID:            resourceID,
		Title:         title,
		Slug:          slug,
		Summary:       summary,
		Category:      categoryDTO,
		AvatarURL:     avatarURL,
		AvatarAssetID: req.AvatarID,
		Tags:          tags,
		AuthorName:    author,
		PublishedAt:   published,
		Status:        status,
		Blocks:        blockJSON,
	})
}

func (s *Server) UpdateResource(w http.ResponseWriter, r *http.Request) {
	userID := CurrentUserID(r)
	resourceID := chi.URLParam(r, "resourceId")
	var req ResourceCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	row := struct {
		AuthorID    string     `db:"author_id"`
		Status      string     `db:"status"`
		Slug        string     `db:"slug"`
		PublishedAt *time.Time `db:"published_at"`
	}{}
	if err := s.DB.Get(&row, `SELECT author_id, status, slug, published_at FROM resource_entries WHERE id = $1`, resourceID); err != nil {
		WriteError(w, http.StatusNotFound, "Resource not found")
		return
	}
	canManage := hasRole(CurrentRoles(r), "ADMIN")
	if !canManage && row.AuthorID != userID {
		WriteError(w, http.StatusNotFound, "Resursa nu a fost găsită.")
		return
	}
	categoryCode := strings.TrimSpace(req.CategoryCode)
	title := strings.TrimSpace(req.Title)
	summary := strings.TrimSpace(req.Summary)
	if categoryCode == "" || title == "" || summary == "" {
		WriteError(w, http.StatusBadRequest, "Titlul și descrierea sunt obligatorii.")
		return
	}
	var categoryExists bool
	_ = s.DB.Get(&categoryExists, `SELECT EXISTS(SELECT 1 FROM resource_categories WHERE code = $1)`, categoryCode)
	if !categoryExists {
		WriteError(w, http.StatusBadRequest, "Categoria selectată nu există.")
		return
	}
	blocks, err := services.ValidateBlocks(req.Blocks)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	blockJSON, _ := json.Marshal(blocks)
	tags := services.CleanTags(req.Tags)
	tagsJSON, _ := json.Marshal(tags)
	status := req.Status
	if status == "" {
		status = row.Status
	}
	now := time.Now().UTC()
	publishedAt := row.PublishedAt
	if status == "PUBLISHED" {
		if publishedAt == nil {
			publishedAt = &now
		}
	} else {
		publishedAt = nil
	}
	_, err = s.DB.Exec(`
UPDATE resource_entries
SET category_code = $2, title = $3, summary = $4, avatar_media_id = $5, tags = $6, content = $7, status = $8, published_at = $9, updated_at = $10
WHERE id = $1
`, resourceID, categoryCode, title, summary, req.AvatarID, tagsJSON, blockJSON, status, publishedAt, now)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	categoryDTO := s.fetchCategory(categoryCode)
	author := s.authorDisplayName(userID)
	var published *string
	if publishedAt != nil {
		formatted := publishedAt.UTC().Format(time.RFC3339)
		published = &formatted
	}
	var avatarURL *string
	if req.AvatarID != nil {
		url := services.BuildAssetURL(*req.AvatarID)
		avatarURL = &url
	}
	WriteJSON(w, http.StatusOK, ResourceDetailDTO{
		ID:            resourceID,
		Title:         title,
		Slug:          row.Slug,
		Summary:       summary,
		Category:      categoryDTO,
		AvatarURL:     avatarURL,
		AvatarAssetID: req.AvatarID,
		Tags:          tags,
		AuthorName:    author,
		PublishedAt:   published,
		Status:        status,
		Blocks:        blockJSON,
	})
}

func (s *Server) DeleteResource(w http.ResponseWriter, r *http.Request) {
	userID := CurrentUserID(r)
	resourceID := chi.URLParam(r, "resourceId")
	row := struct {
		AuthorID string `db:"author_id"`
	}{}
	if err := s.DB.Get(&row, `SELECT author_id FROM resource_entries WHERE id = $1`, resourceID); err != nil {
		WriteError(w, http.StatusNotFound, "Resource not found")
		return
	}
	canManage := hasRole(CurrentRoles(r), "ADMIN")
	if !canManage && row.AuthorID != userID {
		WriteError(w, http.StatusNotFound, "Resursa nu a fost găsită.")
		return
	}
	_, _ = s.DB.Exec(`DELETE FROM resource_entries WHERE id = $1`, resourceID)
	w.WriteHeader(http.StatusNoContent)
}

func hasRole(roles []string, role string) bool {
	role = strings.ToUpper(role)
	for _, r := range roles {
		if strings.ToUpper(r) == role {
			return true
		}
	}
	return false
}
