package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"fizicamd-backend-go/internal/services"

	"github.com/google/uuid"
)

type SearchResultItem struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Slug     string  `json:"slug"`
	Href     *string `json:"href"`
	Type     string  `json:"type"`
	ParentID *string `json:"parentId"`
}

type SearchResponse struct {
	Items []SearchResultItem `json:"items"`
}

type VisitRequest struct {
	Path     *string `json:"path"`
	Referrer *string `json:"referrer"`
}

type VisitCountResponse struct {
	Total int `json:"total"`
}

func (s *Server) PublicSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	term := services.CleanSearchTerm(query)
	if term == "" {
		WriteJSON(w, http.StatusOK, SearchResponse{Items: []SearchResultItem{}})
		return
	}
	rows := []struct {
		ID    string `db:"id"`
		Title string `db:"title"`
		Slug  string `db:"slug"`
	}{}
	like := "%" + strings.ToLower(term) + "%"
	if err := s.DB.Select(&rows, `
SELECT id, title, slug
FROM resource_entries
WHERE status = 'PUBLISHED'
  AND (lower(title) LIKE $1 OR lower(summary) LIKE $1 OR tags::text ILIKE $1)
ORDER BY published_at DESC
LIMIT 20
`, like); err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	items := make([]SearchResultItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, SearchResultItem{
			ID:    row.ID,
			Title: row.Title,
			Slug:  row.Slug,
			Type:  "RESOURCE",
		})
	}
	WriteJSON(w, http.StatusOK, SearchResponse{Items: items})
}

func (s *Server) TrackVisit(w http.ResponseWriter, r *http.Request) {
	var req VisitRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	ip := resolveClientIP(r)
	ua := trimString(r.Header.Get("User-Agent"), 512)
	path := trimString(ptrToString(req.Path), 255)
	ref := trimString(ptrToString(req.Referrer), 512)
	_, _ = s.DB.Exec(`
INSERT INTO site_visits (id, ip_address, user_agent, path, referrer, created_at)
VALUES ($1,$2,$3,$4,$5,$6)
`, uuid.NewString(), nullIfEmpty(ip), nullIfEmpty(ua), nullIfEmpty(path), nullIfEmpty(ref), time.Now().UTC())
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) VisitCount(w http.ResponseWriter, r *http.Request) {
	var total int
	_ = s.DB.Get(&total, `SELECT count(*) FROM site_visits`)
	WriteJSON(w, http.StatusOK, VisitCountResponse{Total: total})
}

func resolveClientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	return r.RemoteAddr
}

func trimString(value string, maxLen int) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) > maxLen {
		return trimmed[:maxLen]
	}
	return trimmed
}

func nullIfEmpty(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func ptrToString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
