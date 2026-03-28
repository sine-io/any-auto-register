package sqliteadapter

import (
	"database/sql"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func Open(rawURL string) (*sql.DB, error) {
	path := strings.TrimSpace(rawURL)
	path = strings.TrimPrefix(path, "sqlite:////")
	path = strings.TrimPrefix(path, "sqlite:///")
	path = strings.TrimPrefix(path, "sqlite://")
	return sql.Open("sqlite3", path)
}
