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

type CategoryUpsertRequest struct {
	Label      string `json:"label"`
	Group      string `json:"group"`
	SortOrder  *int   `json:"sortOrder"`
	GroupOrder *int   `json:"groupOrder"`
}

type GroupUpdateRequest struct {
	Label      string `json:"label"`
	GroupOrder *int   `json:"groupOrder"`
}

func (s *Server) ListCategories(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req CategoryUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	label, err := services.NormalizeRequired(req.Label, "Denumirea categoriei este obligatorie.")
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	groupLabel, err := services.NormalizeRequired(req.Group, "Denumirea grupului este obligatorie.")
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	code, err := services.ResolveCategoryCode(s.DB, label)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	groupOrder := req.GroupOrder
	if groupOrder == nil {
		_ = s.DB.Get(&groupOrder, `SELECT group_order FROM resource_categories WHERE group_label = $1 LIMIT 1`, groupLabel)
		if groupOrder == nil {
			var maxGroup int
			_ = s.DB.Get(&maxGroup, `SELECT COALESCE(MAX(group_order), 0) FROM resource_categories`)
			value := maxGroup + 1
			groupOrder = &value
		}
	}
	sortOrder := req.SortOrder
	if sortOrder == nil {
		var maxSort int
		_ = s.DB.Get(&maxSort, `SELECT COALESCE(MAX(sort_order), 0) FROM resource_categories WHERE group_label = $1`, groupLabel)
		value := maxSort + 1
		sortOrder = &value
	}
	_, err = s.DB.Exec(`
INSERT INTO resource_categories (id, code, label, group_label, sort_order, group_order, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7)
`, uuid.NewString(), code, label, groupLabel, *sortOrder, *groupOrder, time.Now().UTC())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	WriteJSON(w, http.StatusOK, CategoryDTO{Code: code, Label: label, Group: groupLabel, SortOrder: *sortOrder, GroupOrder: *groupOrder})
}

func (s *Server) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	var req CategoryUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	label, err := services.NormalizeRequired(req.Label, "Denumirea categoriei este obligatorie.")
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	groupLabel, err := services.NormalizeRequired(req.Group, "Denumirea grupului este obligatorie.")
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	row := struct {
		GroupLabel string `db:"group_label"`
		GroupOrder int    `db:"group_order"`
		SortOrder  int    `db:"sort_order"`
	}{}
	if err := s.DB.Get(&row, `SELECT group_label, group_order, sort_order FROM resource_categories WHERE code = $1`, code); err != nil {
		WriteError(w, http.StatusNotFound, "Categoria nu există.")
		return
	}
	movingGroup := row.GroupLabel != groupLabel
	groupOrder := req.GroupOrder
	if groupOrder == nil {
		if movingGroup {
			var existing *int
			_ = s.DB.Get(&existing, `SELECT group_order FROM resource_categories WHERE group_label = $1 LIMIT 1`, groupLabel)
			if existing != nil {
				groupOrder = existing
			} else {
				var maxGroup int
				_ = s.DB.Get(&maxGroup, `SELECT COALESCE(MAX(group_order), 0) FROM resource_categories`)
				value := maxGroup + 1
				groupOrder = &value
			}
		} else {
			value := row.GroupOrder
			groupOrder = &value
		}
	}
	sortOrder := req.SortOrder
	if sortOrder == nil {
		if movingGroup {
			var maxSort int
			_ = s.DB.Get(&maxSort, `SELECT COALESCE(MAX(sort_order), 0) FROM resource_categories WHERE group_label = $1`, groupLabel)
			value := maxSort + 1
			sortOrder = &value
		} else {
			value := row.SortOrder
			sortOrder = &value
		}
	}
	_, err = s.DB.Exec(`
UPDATE resource_categories
SET label = $2, group_label = $3, sort_order = $4, group_order = $5
WHERE code = $1
`, code, label, groupLabel, *sortOrder, *groupOrder)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	WriteJSON(w, http.StatusOK, CategoryDTO{Code: code, Label: label, Group: groupLabel, SortOrder: *sortOrder, GroupOrder: *groupOrder})
}

func (s *Server) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	var exists bool
	_ = s.DB.Get(&exists, `SELECT EXISTS(SELECT 1 FROM resource_categories WHERE code = $1)`, code)
	if !exists {
		WriteError(w, http.StatusNotFound, "Categoria nu există.")
		return
	}
	var hasResources bool
	_ = s.DB.Get(&hasResources, `SELECT EXISTS(SELECT 1 FROM resource_entries WHERE category_code = $1)`, code)
	if hasResources {
		WriteError(w, http.StatusBadRequest, "Nu poți șterge această categorie deoarece există resurse asociate.")
		return
	}
	_, _ = s.DB.Exec(`DELETE FROM resource_categories WHERE code = $1`, code)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) UpdateGroupLabel(w http.ResponseWriter, r *http.Request) {
	groupLabel := strings.TrimSpace(chi.URLParam(r, "groupLabel"))
	var req GroupUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	current, err := services.NormalizeRequired(groupLabel, "Grupul selectat este invalid.")
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	newLabel, err := services.NormalizeRequired(req.Label, "Denumirea grupului este obligatorie.")
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	rows := []struct {
		Code       string `db:"code"`
		Label      string `db:"label"`
		GroupLabel string `db:"group_label"`
		SortOrder  int    `db:"sort_order"`
		GroupOrder int    `db:"group_order"`
	}{}
	if err := s.DB.Select(&rows, `SELECT code, label, group_label, sort_order, group_order FROM resource_categories WHERE group_label = $1`, current); err != nil || len(rows) == 0 {
		WriteError(w, http.StatusNotFound, "Grupul nu există.")
		return
	}
	groupOrder := req.GroupOrder
	if groupOrder == nil {
		value := rows[0].GroupOrder
		groupOrder = &value
	}
	_, _ = s.DB.Exec(`UPDATE resource_categories SET group_label = $1, group_order = $2 WHERE group_label = $3`, newLabel, *groupOrder, current)

	updated := make([]CategoryDTO, 0, len(rows))
	for _, row := range rows {
		updated = append(updated, CategoryDTO{Code: row.Code, Label: row.Label, Group: newLabel, SortOrder: row.SortOrder, GroupOrder: *groupOrder})
	}
	WriteJSON(w, http.StatusOK, updated)
}
