package sqliteadapter

import (
	"context"
	"database/sql"
	"strings"
)

type ConfigRepository struct {
	db *sql.DB
}

func NewConfigRepository(db *sql.DB) ConfigRepository {
	return ConfigRepository{db: db}
}

func (r ConfigRepository) GetAll(ctx context.Context, keys []string) (map[string]string, error) {
	items := make(map[string]string, len(keys))
	if len(keys) == 0 {
		return items, nil
	}

	rows, err := r.db.QueryContext(ctx, `SELECT key, value FROM configs`)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return items, nil
		}
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		items[key] = value
	}
	return items, rows.Err()
}
