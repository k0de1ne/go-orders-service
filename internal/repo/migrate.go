package repo

import (
	"database/sql"
	"os"
	"path/filepath"
	"sort"
)

func RunMigrations(db *sql.DB, migrationsPath string) error {
	files, err := os.ReadDir(migrationsPath)
	if err != nil {
		return err
	}

	var sqlFiles []string
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".sql" {
			sqlFiles = append(sqlFiles, f.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, file := range sqlFiles {
		content, err := os.ReadFile(filepath.Join(migrationsPath, file))
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(content)); err != nil {
			return err
		}
	}
	return nil
}
