package httpapi

import (
	"time"

	"fizicamd-backend-go/internal/services"

	"github.com/jmoiron/sqlx"
)

type ProfileDTO struct {
	FirstName  *string `json:"firstName,omitempty"`
	LastName   *string `json:"lastName,omitempty"`
	BirthDate  *string `json:"birthDate,omitempty"`
	Gender     *string `json:"gender,omitempty"`
	Phone      *string `json:"phone,omitempty"`
	School     *string `json:"school,omitempty"`
	GradeLevel *string `json:"gradeLevel,omitempty"`
	Bio        *string `json:"bio,omitempty"`
	AvatarURL  *string `json:"avatarUrl,omitempty"`
}

type UserDTO struct {
	ID          string      `json:"id"`
	Email       string      `json:"email"`
	Status      string      `json:"status"`
	Role        string      `json:"role"`
	Roles       []string    `json:"roles"`
	Profile     *ProfileDTO `json:"profile,omitempty"`
	LastLoginAt *time.Time  `json:"lastLoginAt,omitempty"`
}

func buildUserDTO(db *sqlx.DB, userID string) (*UserDTO, error) {
	row := struct {
		ID         string     `db:"id"`
		Email      string     `db:"email"`
		Status     string     `db:"status"`
		LastLogin  *time.Time `db:"last_login_at"`
		FirstName  *string    `db:"first_name"`
		LastName   *string    `db:"last_name"`
		BirthDate  *time.Time `db:"birth_date"`
		Gender     *string    `db:"gender"`
		Phone      *string    `db:"phone"`
		School     *string    `db:"school"`
		GradeLevel *string    `db:"grade_level"`
		Bio        *string    `db:"bio"`
		AvatarID   *string    `db:"avatar_media_id"`
	}{}
	if err := db.Get(&row, `
SELECT u.id, u.email, u.status, u.last_login_at,
       p.first_name, p.last_name, p.birth_date, p.gender, p.phone, p.school, p.grade_level, p.bio, p.avatar_media_id
FROM users u
LEFT JOIN user_profiles p ON p.user_id = u.id
WHERE u.id = $1
`, userID); err != nil {
		return nil, err
	}
	roles := []string{}
	if err := db.Select(&roles, `
SELECT r.code FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = $1
ORDER BY r.code
`, userID); err != nil {
		return nil, err
	}
	primary := "STUDENT"
	if len(roles) > 0 {
		primary = roles[0]
	}
	var birthStr *string
	if row.BirthDate != nil {
		formatted := row.BirthDate.Format("2006-01-02")
		birthStr = &formatted
	}
	var avatarURL *string
	if row.AvatarID != nil {
		url := services.BuildAssetURL(*row.AvatarID)
		avatarURL = &url
	}
	profile := (*ProfileDTO)(nil)
	if row.FirstName != nil || row.LastName != nil || row.Phone != nil || row.School != nil || row.GradeLevel != nil || row.Bio != nil || row.AvatarID != nil || row.Gender != nil || row.BirthDate != nil {
		profile = &ProfileDTO{
			FirstName:  row.FirstName,
			LastName:   row.LastName,
			BirthDate:  birthStr,
			Gender:     row.Gender,
			Phone:      row.Phone,
			School:     row.School,
			GradeLevel: row.GradeLevel,
			Bio:        row.Bio,
			AvatarURL:  avatarURL,
		}
	}
	return &UserDTO{
		ID:          row.ID,
		Email:       row.Email,
		Status:      row.Status,
		Role:        primary,
		Roles:       roles,
		Profile:     profile,
		LastLoginAt: row.LastLogin,
	}, nil
}
