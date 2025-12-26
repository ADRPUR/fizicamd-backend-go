package migrations

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
)

type migration struct {
	Name string
	Path string
}

func Apply(db *sqlx.DB, dir string) error {
	if err := ensureTable(db); err != nil {
		return err
	}
	migs, err := listMigrations(dir)
	if err != nil {
		return err
	}
	applied, err := appliedMigrations(db)
	if err != nil {
		return err
	}
	for _, mig := range migs {
		version := parseVersion(mig.Name)
		if applied.names[mig.Name] || (version != "" && applied.versions[version]) {
			continue
		}
		if err := applyMigration(db, mig); err != nil {
			return err
		}
	}
	return nil
}

func ensureTable(db *sqlx.DB) error {
	var exists bool
	if err := db.Get(&exists, `
SELECT EXISTS(
  SELECT 1 FROM information_schema.tables
  WHERE table_schema = 'public' AND table_name = 'schema_migrations'
)`); err != nil {
		return err
	}
	if !exists {
		_, err := db.Exec(`
CREATE TABLE schema_migrations (
  id SERIAL PRIMARY KEY,
  version TEXT NULL,
  name TEXT NULL,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_schema_migrations_name ON schema_migrations(name) WHERE name IS NOT NULL`)
		if err != nil {
			return err
		}
		_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_schema_migrations_version ON schema_migrations(version) WHERE version IS NOT NULL`)
		return err
	}
	_, err := db.Exec(`ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS name TEXT`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS version TEXT`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_schema_migrations_name ON schema_migrations(name) WHERE name IS NOT NULL`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_schema_migrations_version ON schema_migrations(version) WHERE version IS NOT NULL`)
	return err
}

func listMigrations(dir string) ([]migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	migs := make([]migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		migs = append(migs, migration{
			Name: name,
			Path: filepath.Join(dir, name),
		})
	}
	sort.Slice(migs, func(i, j int) bool {
		iVersion, iOk := parseVersionNumber(migs[i].Name)
		jVersion, jOk := parseVersionNumber(migs[j].Name)
		switch {
		case iOk && jOk && iVersion != jVersion:
			return iVersion < jVersion
		case iOk != jOk:
			return iOk
		default:
			return migs[i].Name < migs[j].Name
		}
	})
	return migs, nil
}

type appliedSet struct {
	names    map[string]bool
	versions map[string]bool
}

func appliedMigrations(db *sqlx.DB) (appliedSet, error) {
	hasName, hasVersion, err := schemaColumns(db)
	if err != nil {
		return appliedSet{}, err
	}
	names := map[string]bool{}
	versions := map[string]bool{}
	if hasName {
		rows := []string{}
		if err := db.Select(&rows, `SELECT name FROM schema_migrations WHERE name IS NOT NULL`); err != nil {
			return appliedSet{}, err
		}
		for _, name := range rows {
			names[name] = true
		}
	}
	if hasVersion {
		rows := []string{}
		if err := db.Select(&rows, `SELECT version FROM schema_migrations WHERE version IS NOT NULL`); err != nil {
			return appliedSet{}, err
		}
		for _, version := range rows {
			versions[version] = true
		}
	}
	return appliedSet{names: names, versions: versions}, nil
}

func applyMigration(db *sqlx.DB, mig migration) error {
	content, err := os.ReadFile(mig.Path)
	if err != nil {
		return err
	}
	if _, err := db.Exec(string(content)); err != nil {
		return fmt.Errorf("apply %s: %w", mig.Name, err)
	}
	hasName, hasVersion, err := schemaColumns(db)
	if err != nil {
		return err
	}
	version := parseVersion(mig.Name)
	switch {
	case hasName && hasVersion:
		_, err = db.Exec(`INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`, nullIfEmpty(version), mig.Name)
	case hasVersion:
		_, err = db.Exec(`INSERT INTO schema_migrations (version) VALUES ($1)`, nullIfEmpty(version))
	default:
		_, err = db.Exec(`INSERT INTO schema_migrations (name) VALUES ($1)`, mig.Name)
	}
	return err
}

func CopyMigrations(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())
		if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, content, fs.FileMode(0644))
}

func schemaColumns(db *sqlx.DB) (bool, bool, error) {
	rows := []string{}
	if err := db.Select(&rows, `
SELECT column_name
FROM information_schema.columns
WHERE table_schema = 'public' AND table_name = 'schema_migrations'
`); err != nil {
		return false, false, err
	}
	hasName := false
	hasVersion := false
	for _, name := range rows {
		if name == "name" {
			hasName = true
		}
		if name == "version" {
			hasVersion = true
		}
	}
	return hasName, hasVersion, nil
}

func parseVersion(name string) string {
	if !strings.HasPrefix(name, "V") {
		return ""
	}
	parts := strings.SplitN(name[1:], "__", 2)
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func parseVersionNumber(name string) (int, bool) {
	raw := parseVersion(name)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return value, true
}

func nullIfEmpty(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
