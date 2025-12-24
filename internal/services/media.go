package services

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const (
	BucketUsers     = "users"
	BucketResources = "resources"
)

func EnsureStoragePath(base string, bucket string) (string, error) {
	path := filepath.Join(base, bucket)
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", err
	}
	return path, nil
}

func SaveMediaAsset(db *sqlx.DB, basePath, bucket, contentType, filename, mediaType, ownerID string, body io.Reader) (string, string, error) {
	assetID := uuid.NewString()
	storageKey := assetID
	bucketPath, err := EnsureStoragePath(basePath, bucket)
	if err != nil {
		return "", "", err
	}
	targetPath := filepath.Join(bucketPath, storageKey)

	file, err := os.Create(targetPath)
	if err != nil {
		return "", "", err
	}
	hasher := sha256.New()
	writer := io.MultiWriter(file, hasher)
	size, err := io.Copy(writer, body)
	_ = file.Close()
	if err != nil {
		return "", "", err
	}
	if size == 0 {
		_ = os.Remove(targetPath)
		return "", "", ErrBadRequest("Fișierul este gol.")
	}
	sha := hex.EncodeToString(hasher.Sum(nil))

	_, err = db.Exec(`
INSERT INTO media_assets (
  id, owner_user_id, bucket, storage_key, filename, description, type,
  content_type, size_bytes, sha256, access_policy, status, metadata, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'{}', $13, $13)
`, assetID, ownerID, bucket, storageKey, filename, nil, mediaType, contentType, size, sha, "PRIVATE", "READY", time.Now().UTC())
	if err != nil {
		_ = os.Remove(targetPath)
		return "", "", err
	}
	return assetID, BuildAssetURL(assetID), nil
}

func BuildAssetURL(assetID string) string {
	return "/media/assets/" + assetID + "/content"
}

func DeleteAsset(db *sqlx.DB, basePath string, assetID string) error {
	var bucket string
	var storageKey string
	err := db.Get(&bucket, `SELECT bucket FROM media_assets WHERE id = $1`, assetID)
	if err != nil {
		return nil
	}
	err = db.Get(&storageKey, `SELECT storage_key FROM media_assets WHERE id = $1`, assetID)
	if err != nil {
		return nil
	}
	_, _ = db.Exec(`DELETE FROM media_assets WHERE id = $1`, assetID)
	path := filepath.Join(basePath, bucket, storageKey)
	_ = os.Remove(path)
	return nil
}
