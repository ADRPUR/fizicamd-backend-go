package httpapi

import (
	"net/http"
	"strings"

	"fizicamd-backend-go/internal/services"
)

func (s *Server) fetchCategory(code string) *CategoryDTO {
	row := struct {
		Code       string `db:"code"`
		Label      string `db:"label"`
		GroupLabel string `db:"group_label"`
		SortOrder  int    `db:"sort_order"`
		GroupOrder int    `db:"group_order"`
	}{}
	if err := s.DB.Get(&row, `SELECT code, label, group_label, sort_order, group_order FROM resource_categories WHERE code = $1`, code); err != nil {
		return nil
	}
	return &CategoryDTO{
		Code:       row.Code,
		Label:      row.Label,
		Group:      row.GroupLabel,
		SortOrder:  row.SortOrder,
		GroupOrder: row.GroupOrder,
	}
}

func (s *Server) authorDisplayName(authorID string) string {
	row := struct {
		Email     string  `db:"email"`
		FirstName *string `db:"first_name"`
		LastName  *string `db:"last_name"`
	}{}
	if err := s.DB.Get(&row, `
SELECT u.email, p.first_name, p.last_name
FROM users u
LEFT JOIN user_profiles p ON p.user_id = u.id
WHERE u.id = $1
`, authorID); err != nil {
		return "Profesor"
	}
	if (row.FirstName != nil && strings.TrimSpace(*row.FirstName) != "") || (row.LastName != nil && strings.TrimSpace(*row.LastName) != "") {
		name := strings.TrimSpace(strings.TrimSpace(deref(row.FirstName)) + " " + strings.TrimSpace(deref(row.LastName)))
		if name != "" {
			return name
		}
	}
	if row.Email != "" {
		return row.Email
	}
	return "Profesor"
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func mapServiceError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if serr, ok := err.(services.ServiceError); ok {
		WriteError(w, serr.Status, serr.Message)
		return true
	}
	return false
}
