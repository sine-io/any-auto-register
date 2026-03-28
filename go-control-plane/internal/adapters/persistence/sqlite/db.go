package sqliteadapter

import (
	"database/sql"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func Open(rawURL string) (*sql.DB, error) {
	path := strings.TrimSpace(rawURL)
	if strings.HasPrefix(path, "sqlite:////") {
		path = "/" + strings.TrimPrefix(path, "sqlite:////")
	} else {
		path = strings.TrimPrefix(path, "sqlite:///")
	}
	path = strings.TrimPrefix(path, "sqlite:///")
	path = strings.TrimPrefix(path, "sqlite://")
	return sql.Open("sqlite3", path)
}
