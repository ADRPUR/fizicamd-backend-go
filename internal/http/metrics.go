package httpapi

import (
	"net/http"

	"fizicamd-backend-go/internal/services"

	"github.com/gorilla/websocket"
)

type MetricsHistoryResponse struct {
	Items []services.MetricSample `json:"items"`
}

func (s *Server) MetricsHistory(w http.ResponseWriter, r *http.Request) {
	limit := parseInt(r.URL.Query().Get("limit"), 120)
	if limit > 500 {
		limit = 500
	}
	items, err := services.LatestMetrics(s.DB, limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	WriteJSON(w, http.StatusOK, MetricsHistoryResponse{Items: items})
}

func (s *Server) MetricsSocket(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("token")
	if query == "" {
		WriteError(w, http.StatusUnauthorized, "Authentication failed")
		return
	}
	token, claims, err := s.Tokens.ParseToken(query)
	if err != nil || !token.Valid || claims["typ"] != "access" {
		WriteError(w, http.StatusUnauthorized, "Authentication failed")
		return
	}
	roles := []string{}
	if rawRoles, ok := claims["roles"].([]interface{}); ok {
		for _, r := range rawRoles {
			if srole, ok := r.(string); ok {
				roles = append(roles, srole)
			}
		}
	}
	if !hasRole(roles, "ADMIN") {
		WriteError(w, http.StatusForbidden, "Not allowed")
		return
	}
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	s.MetricsHub.Add(conn)
	defer func() {
		s.MetricsHub.Remove(conn)
		_ = conn.Close()
	}()
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
