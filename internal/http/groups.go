package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"fizicamd-backend-go/internal/services"

	"github.com/go-chi/chi/v5"
)

type CreateGroupRequest struct {
	Name  string `json:"name"`
	Grade *int   `json:"grade"`
	Year  *int   `json:"year"`
}

type UpdateGroupRequest struct {
	Name  *string `json:"name"`
	Grade *int    `json:"grade"`
	Year  *int    `json:"year"`
}

type AddMemberRequest struct {
	UserID     string `json:"userId"`
	MemberRole string `json:"memberRole"`
}

type MemberResponse struct {
	UserID     string `json:"userId"`
	Email      string `json:"email"`
	MemberRole string `json:"memberRole"`
}

type GroupResponse struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Grade     *int             `json:"grade"`
	Year      *int             `json:"year"`
	CreatedAt *time.Time       `json:"createdAt"`
	Members   []MemberResponse `json:"members"`
}

func (s *Server) AdminCreateGroup(w http.ResponseWriter, r *http.Request) {
	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		WriteError(w, http.StatusBadRequest, "Name is required")
		return
	}
	_, err := services.CreateGroup(s.DB, req.Name, req.Grade, req.Year)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) AdminUpdateGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupId")
	var req UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	if err := services.UpdateGroup(s.DB, groupID, req.Name, req.Grade, req.Year); err != nil {
		WriteError(w, http.StatusNotFound, "Group not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) AdminDeleteGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupId")
	if err := services.DeleteGroup(s.DB, groupID); err != nil {
		WriteError(w, http.StatusNotFound, "Group not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) AdminAddMember(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupId")
	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	if req.UserID == "" || req.MemberRole == "" {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	if err := services.AddMember(s.DB, groupID, req.UserID, req.MemberRole); err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) AdminRemoveMember(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupId")
	userID := chi.URLParam(r, "userId")
	if err := services.RemoveMember(s.DB, groupID, userID); err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) AdminGetGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupId")
	response, err := s.getGroup(groupID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "Group not found")
		return
	}
	WriteJSON(w, http.StatusOK, response)
}

func (s *Server) TeacherGroups(w http.ResponseWriter, r *http.Request) {
	userID := CurrentUserID(r)
	groups, err := s.userGroups(userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	WriteJSON(w, http.StatusOK, groups)
}

func (s *Server) TeacherGetGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupId")
	userID := CurrentUserID(r)
	if !s.isGroupMember(groupID, userID) {
		WriteError(w, http.StatusForbidden, "Not allowed")
		return
	}
	response, err := s.getGroup(groupID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "Group not found")
		return
	}
	WriteJSON(w, http.StatusOK, response)
}

func (s *Server) TeacherUpdateGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupId")
	userID := CurrentUserID(r)
	if !s.isTeacherInGroup(groupID, userID) {
		WriteError(w, http.StatusForbidden, "Not allowed")
		return
	}
	var req UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	if err := services.UpdateGroup(s.DB, groupID, req.Name, req.Grade, req.Year); err != nil {
		WriteError(w, http.StatusNotFound, "Group not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) TeacherAddMember(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupId")
	userID := CurrentUserID(r)
	if !s.isTeacherInGroup(groupID, userID) {
		WriteError(w, http.StatusForbidden, "Teacher can manage only own groups")
		return
	}
	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid payload")
		return
	}
	role := strings.ToUpper(strings.TrimSpace(req.MemberRole))
	if role == "TEACHER" {
		WriteError(w, http.StatusForbidden, "Teacher cannot grant TEACHER role")
		return
	}
	if err := services.AddMember(s.DB, groupID, req.UserID, role); err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) TeacherRemoveMember(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupId")
	userID := CurrentUserID(r)
	memberID := chi.URLParam(r, "userId")
	if !s.isTeacherInGroup(groupID, userID) {
		WriteError(w, http.StatusForbidden, "Teacher can manage only own groups")
		return
	}
	role := s.memberRole(groupID, memberID)
	if role != "STUDENT" {
		WriteError(w, http.StatusForbidden, "Teacher can remove only students")
		return
	}
	_ = services.RemoveMember(s.DB, groupID, memberID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) StudentGroups(w http.ResponseWriter, r *http.Request) {
	userID := CurrentUserID(r)
	groups, err := s.userGroups(userID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	WriteJSON(w, http.StatusOK, groups)
}

func (s *Server) StudentGetGroup(w http.ResponseWriter, r *http.Request) {
	groupID := chi.URLParam(r, "groupId")
	userID := CurrentUserID(r)
	if !s.isGroupMember(groupID, userID) {
		WriteError(w, http.StatusForbidden, "Not allowed")
		return
	}
	response, err := s.getGroup(groupID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "Group not found")
		return
	}
	WriteJSON(w, http.StatusOK, response)
}

func (s *Server) getGroup(groupID string) (GroupResponse, error) {
	group := struct {
		ID        string     `db:"id"`
		Name      string     `db:"name"`
		Grade     *int       `db:"grade"`
		Year      *int       `db:"year"`
		CreatedAt *time.Time `db:"created_at"`
	}{}
	if err := s.DB.Get(&group, `SELECT id, name, grade, year, created_at FROM groups WHERE id = $1`, groupID); err != nil {
		return GroupResponse{}, err
	}
	members := []struct {
		UserID     string `db:"user_id"`
		Email      string `db:"email"`
		MemberRole string `db:"member_role"`
	}{}
	_ = s.DB.Select(&members, `
SELECT gm.user_id, u.email, gm.member_role
FROM group_members gm
JOIN users u ON u.id = gm.user_id
WHERE gm.group_id = $1
`, groupID)
	items := make([]MemberResponse, 0, len(members))
	for _, member := range members {
		items = append(items, MemberResponse{UserID: member.UserID, Email: member.Email, MemberRole: member.MemberRole})
	}
	return GroupResponse{ID: group.ID, Name: group.Name, Grade: group.Grade, Year: group.Year, CreatedAt: group.CreatedAt, Members: items}, nil
}

func (s *Server) userGroups(userID string) ([]GroupResponse, error) {
	ids := []string{}
	if err := s.DB.Select(&ids, `SELECT DISTINCT group_id FROM group_members WHERE user_id = $1`, userID); err != nil {
		return nil, err
	}
	groups := make([]GroupResponse, 0, len(ids))
	for _, id := range ids {
		group, err := s.getGroup(id)
		if err == nil {
			groups = append(groups, group)
		}
	}
	return groups, nil
}

func (s *Server) isGroupMember(groupID, userID string) bool {
	var exists bool
	_ = s.DB.Get(&exists, `SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2)`, groupID, userID)
	return exists
}

func (s *Server) isTeacherInGroup(groupID, userID string) bool {
	var exists bool
	_ = s.DB.Get(&exists, `SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = $1 AND user_id = $2 AND member_role = 'TEACHER')`, groupID, userID)
	return exists
}

func (s *Server) memberRole(groupID, userID string) string {
	var role string
	_ = s.DB.Get(&role, `SELECT member_role FROM group_members WHERE group_id = $1 AND user_id = $2`, groupID, userID)
	return role
}
