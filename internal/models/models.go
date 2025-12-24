package models

import "time"

type User struct {
	ID           string     `db:"id"`
	Email        string     `db:"email"`
	PasswordHash string     `db:"password_hash"`
	Status       string     `db:"status"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
	LastLoginAt  *time.Time `db:"last_login_at"`
	LastSeenAt   *time.Time `db:"last_seen_at"`
}

type UserProfile struct {
	UserID      string     `db:"user_id"`
	FirstName   *string    `db:"first_name"`
	LastName    *string    `db:"last_name"`
	BirthDate   *time.Time `db:"birth_date"`
	Gender      *string    `db:"gender"`
	Phone       *string    `db:"phone"`
	School      *string    `db:"school"`
	GradeLevel  *string    `db:"grade_level"`
	Bio         *string    `db:"bio"`
	AvatarMedia *string    `db:"avatar_media_id"`
}

type Role struct {
	ID   string `db:"id"`
	Code string `db:"code"`
}

type MediaAsset struct {
	ID          string    `db:"id"`
	OwnerUserID *string   `db:"owner_user_id"`
	Bucket      string    `db:"bucket"`
	StorageKey  string    `db:"storage_key"`
	Filename    *string   `db:"filename"`
	Type        string    `db:"type"`
	ContentType string    `db:"content_type"`
	SizeBytes   int64     `db:"size_bytes"`
	Sha256      *string   `db:"sha256"`
	CreatedAt   time.Time `db:"created_at"`
}

type ResourceCategory struct {
	ID         string    `db:"id"`
	Code       string    `db:"code"`
	Label      string    `db:"label"`
	GroupLabel string    `db:"group_label"`
	SortOrder  int       `db:"sort_order"`
	GroupOrder int       `db:"group_order"`
	CreatedAt  time.Time `db:"created_at"`
}

type ResourceEntry struct {
	ID           string     `db:"id"`
	CategoryCode string     `db:"category_code"`
	AuthorID     string     `db:"author_id"`
	Title        string     `db:"title"`
	Slug         string     `db:"slug"`
	Summary      string     `db:"summary"`
	AvatarMedia  *string    `db:"avatar_media_id"`
	Content      []byte     `db:"content"`
	Tags         []byte     `db:"tags"`
	Status       string     `db:"status"`
	PublishedAt  *time.Time `db:"published_at"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

type ServerMetricSample struct {
	ID                string    `db:"id"`
	CapturedAt        time.Time `db:"captured_at"`
	HeapUsedBytes     int64     `db:"heap_used_bytes"`
	HeapMaxBytes      int64     `db:"heap_max_bytes"`
	SystemMemoryTotal int64     `db:"system_memory_total_bytes"`
	SystemMemoryUsed  int64     `db:"system_memory_used_bytes"`
	DiskTotalBytes    int64     `db:"disk_total_bytes"`
	DiskUsedBytes     int64     `db:"disk_used_bytes"`
	ProcessCpuLoad    float64   `db:"process_cpu_load"`
	SystemCpuLoad     float64   `db:"system_cpu_load"`
}

type Group struct {
	ID        string    `db:"id"`
	Name      string    `db:"name"`
	Grade     *int      `db:"grade"`
	Year      *int      `db:"year"`
	CreatedAt time.Time `db:"created_at"`
}

type GroupMember struct {
	ID         string    `db:"id"`
	GroupID    string    `db:"group_id"`
	UserID     string    `db:"user_id"`
	MemberRole string    `db:"member_role"`
	CreatedAt  time.Time `db:"created_at"`
}

type SiteVisit struct {
	ID        string    `db:"id"`
	IPAddress *string   `db:"ip_address"`
	UserAgent *string   `db:"user_agent"`
	Path      *string   `db:"path"`
	Referrer  *string   `db:"referrer"`
	CreatedAt time.Time `db:"created_at"`
}
