package sqliteadapter

import (
	"context"
	"database/sql"
	"strings"

	viperconfig "go-control-plane/internal/adapters/config/viper"
	domainplatform "go-control-plane/internal/domain/platform"
)

type PlatformRepository struct {
	db       *sql.DB
	fallback []domainplatform.Platform
}

func NewPlatformRepository(db *sql.DB, entries []viperconfig.PlatformEntry) PlatformRepository {
	fallback := make([]domainplatform.Platform, 0, len(entries))
	for _, entry := range entries {
		fallback = append(fallback, domainplatform.Platform{
			Name:               entry.Name,
			DisplayName:        entry.DisplayName,
			Version:            entry.Version,
			SupportedExecutors: entry.SupportedExecutors,
			Available:          entry.Available,
			AvailabilityReason: entry.AvailabilityReason,
		})
	}
	return PlatformRepository{db: db, fallback: fallback}
}

func (r PlatformRepository) List(ctx context.Context) ([]domainplatform.Platform, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT name, display_name, version, supported_executors_json, available, availability_reason
		 FROM platform_manifest
		 ORDER BY name`,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return r.fallback, nil
		}
		return nil, err
	}
	defer rows.Close()

	items := make([]domainplatform.Platform, 0)
	for rows.Next() {
		var item domainplatform.Platform
		var executorsJSON string
		if err := rows.Scan(
			&item.Name,
			&item.DisplayName,
			&item.Version,
			&executorsJSON,
			&item.Available,
			&item.AvailabilityReason,
		); err != nil {
			return nil, err
		}
		item.SupportedExecutors = decodeExecutors(executorsJSON)
		items = append(items, item)
	}
	if len(items) == 0 {
		return r.fallback, rows.Err()
	}
	return items, rows.Err()
}

func decodeExecutors(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{"protocol"}
	}
	raw = strings.Trim(raw, "[]")
	raw = strings.ReplaceAll(raw, "\"", "")
	if raw == "" {
		return []string{"protocol"}
	}
	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			items = append(items, value)
		}
	}
	if len(items) == 0 {
		return []string{"protocol"}
	}
	return items
}
