package services

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var blockTypes = map[string]bool{
	"TEXT":    true,
	"LINK":    true,
	"IMAGE":   true,
	"PDF":     true,
	"FORMULA": true,
}

func Slugify(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return uuid.NewString()
	}
	return slug
}

func ResolveCategoryCode(db *sqlx.DB, label string) (string, error) {
	base := Slugify(label)
	candidate := base
	counter := 2
	for {
		var exists bool
		err := db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM resource_categories WHERE code = $1)`, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = base + "-" + strconv.Itoa(counter)
		counter++
	}
}

func ResolveResourceSlug(db *sqlx.DB, title string) (string, error) {
	base := Slugify(title)
	candidate := base
	counter := 2
	for {
		var exists bool
		err := db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM resource_entries WHERE slug = $1)`, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = base + "-" + strconv.Itoa(counter)
		counter++
	}
}

func CleanTags(tags []string) []string {
	seen := make(map[string]bool)
	cleaned := make([]string, 0, len(tags))
	for _, tag := range tags {
		value := strings.TrimSpace(tag)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		cleaned = append(cleaned, value)
		if len(cleaned) >= 12 {
			break
		}
	}
	return cleaned
}

func NormalizeRequired(value, message string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", errors.New(message)
	}
	return trimmed, nil
}

type Block struct {
	Type    string  `json:"type"`
	Text    *string `json:"text,omitempty"`
	URL     *string `json:"url,omitempty"`
	AssetID *string `json:"assetId,omitempty"`
	Caption *string `json:"caption,omitempty"`
	Title   *string `json:"title,omitempty"`
}

func ValidateBlocks(raw json.RawMessage) ([]Block, error) {
	if len(raw) == 0 {
		return []Block{}, nil
	}
	var blocks []Block
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, err
	}
	cleaned := make([]Block, 0, len(blocks))
	for _, block := range blocks {
		blockType := strings.ToUpper(strings.TrimSpace(block.Type))
		if !blockTypes[blockType] {
			continue
		}
		switch blockType {
		case "TEXT":
			if block.Text != nil {
				text := strings.TrimSpace(*block.Text)
				if text != "" {
					cleaned = append(cleaned, Block{Type: "TEXT", Text: &text, Title: block.Title})
				}
			}
		case "LINK":
			if block.URL != nil {
				url := strings.TrimSpace(*block.URL)
				if url != "" {
					title := block.Title
					if title == nil {
						fallback := "Link"
						title = &fallback
					}
					cleaned = append(cleaned, Block{Type: "LINK", URL: &url, Title: title})
				}
			}
		case "IMAGE", "PDF":
			if block.AssetID == nil || strings.TrimSpace(*block.AssetID) == "" {
				return nil, errors.New("Încărcarea fișierului pentru blocurile media este obligatorie.")
			}
			asset := strings.TrimSpace(*block.AssetID)
			cleaned = append(cleaned, Block{Type: blockType, AssetID: &asset, Caption: block.Caption, Title: block.Title})
		case "FORMULA":
			if block.Text == nil || strings.TrimSpace(*block.Text) == "" {
				return nil, errors.New("Formula nu poate fi goală.")
			}
			text := strings.TrimSpace(*block.Text)
			cleaned = append(cleaned, Block{Type: "FORMULA", Text: &text, Title: block.Title})
		}
	}
	return cleaned, nil
}

func CleanSearchTerm(term string) string {
	re := regexp.MustCompile(`\s+`)
	cleaned := strings.TrimSpace(term)
	cleaned = re.ReplaceAllString(cleaned, " ")
	return cleaned
}
