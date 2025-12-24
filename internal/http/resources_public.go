package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"fizicamd-backend-go/internal/services"

	"github.com/go-chi/chi/v5"
)

func (s *Server) PublicCategories(w http.ResponseWriter, r *http.Request) {
	rows := []struct {
		Code       string `db:"code"`
		Label      string `db:"label"`
		GroupLabel string `db:"group_label"`
		SortOrder  int    `db:"sort_order"`
		GroupOrder int    `db:"group_order"`
	}{}
	if err := s.DB.Select(&rows, `SELECT code, label, group_label, sort_order, group_order FROM resource_categories ORDER BY group_order ASC, sort_order ASC`); err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	items := make([]CategoryDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, CategoryDTO{
			Code:       row.Code,
			Label:      row.Label,
			Group:      row.GroupLabel,
			SortOrder:  row.SortOrder,
			GroupOrder: row.GroupOrder,
		})
	}
	WriteJSON(w, http.StatusOK, items)
}

func (s *Server) PublicResources(w http.ResponseWriter, r *http.Request) {
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	limit := parseInt(r.URL.Query().Get("limit"), 9)
	page := parseInt(r.URL.Query().Get("page"), 1)
	if limit < 1 {
		limit = 9
	}
	offset := (page - 1) * limit

	args := []interface{}{}
	where := "WHERE status = 'PUBLISHED'"
	if category != "" {
		where += " AND category_code = $1"
		args = append(args, category)
	}
	var total int
	if err := s.DB.Get(&total, "SELECT count(*) FROM resource_entries "+where, args...); err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	args = append(args, limit, offset)
	query := `
SELECT id, category_code, author_id, title, slug, summary, avatar_media_id, tags, status, published_at
FROM resource_entries
` + where + `
ORDER BY published_at DESC
LIMIT $%d OFFSET $%d`
	query = fmt.Sprintf(query, len(args)-1, len(args))
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
	WriteJSON(w, http.StatusOK, ResourceListResponse{Items: items, Total: total, Page: page, Size: limit})
}

func (s *Server) PublicResourceDetail(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
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
WHERE slug = $1
`, slug); err != nil {
		WriteError(w, http.StatusNotFound, "Resursa nu există")
		return
	}
	if row.Status != "PUBLISHED" {
		WriteError(w, http.StatusNotFound, "Resursa nu este publică")
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
