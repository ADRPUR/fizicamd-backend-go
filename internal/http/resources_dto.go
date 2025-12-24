package httpapi

import "encoding/json"

type CategoryDTO struct {
	Code       string `json:"code"`
	Label      string `json:"label"`
	Group      string `json:"group"`
	SortOrder  int    `json:"sortOrder"`
	GroupOrder int    `json:"groupOrder"`
}

type ResourceCardDTO struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Slug        string       `json:"slug"`
	Summary     string       `json:"summary"`
	Category    *CategoryDTO `json:"category"`
	AvatarURL   *string      `json:"avatarUrl"`
	Tags        []string     `json:"tags"`
	AuthorName  string       `json:"authorName"`
	PublishedAt *string      `json:"publishedAt"`
	Status      string       `json:"status"`
}

type ResourceDetailDTO struct {
	ID            string          `json:"id"`
	Title         string          `json:"title"`
	Slug          string          `json:"slug"`
	Summary       string          `json:"summary"`
	Category      *CategoryDTO    `json:"category"`
	AvatarURL     *string         `json:"avatarUrl"`
	AvatarAssetID *string         `json:"avatarAssetId"`
	Tags          []string        `json:"tags"`
	AuthorName    string          `json:"authorName"`
	PublishedAt   *string         `json:"publishedAt"`
	Status        string          `json:"status"`
	Blocks        json.RawMessage `json:"blocks"`
}

type ResourceListResponse struct {
	Items []ResourceCardDTO `json:"items"`
	Total int               `json:"total"`
	Page  int               `json:"page"`
	Size  int               `json:"size"`
}
