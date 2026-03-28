package sqliteadapter

import (
	"context"
	"database/sql"
	"net/http"
	"net/url"
	"strings"
	"time"

	domainproxy "go-control-plane/internal/domain/proxy"
)

type ProxyRepository struct {
	db *sql.DB
}

func NewProxyRepository(db *sql.DB) ProxyRepository {
	return ProxyRepository{db: db}
}

func (r ProxyRepository) List(ctx context.Context, _ domainproxy.ListFilter) ([]domainproxy.Proxy, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, url, region, success_count, fail_count, is_active, last_checked FROM proxies ORDER BY id DESC`)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return []domainproxy.Proxy{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	items := make([]domainproxy.Proxy, 0)
	for rows.Next() {
		var item domainproxy.Proxy
		var lastChecked sql.NullString
		if err := rows.Scan(&item.ID, &item.URL, &item.Region, &item.SuccessCount, &item.FailCount, &item.IsActive, &lastChecked); err != nil {
			return nil, err
		}
		if lastChecked.Valid {
			parsed := parseTime(lastChecked.String)
			item.LastChecked = &parsed
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r ProxyRepository) Add(ctx context.Context, rawURL, region string) (int64, error) {
	result, err := r.db.ExecContext(ctx, `INSERT INTO proxies (url, region, success_count, fail_count, is_active) VALUES (?, ?, 0, 0, 1)`, rawURL, region)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r ProxyRepository) BulkAdd(ctx context.Context, proxies []string, region string) (int64, error) {
	var added int64
	for _, rawURL := range proxies {
		value := strings.TrimSpace(rawURL)
		if value == "" {
			continue
		}
		_, err := r.db.ExecContext(ctx, `INSERT OR IGNORE INTO proxies (url, region, success_count, fail_count, is_active) VALUES (?, ?, 0, 0, 1)`, value, region)
		if err != nil {
			return added, err
		}
		res, err := r.db.ExecContext(ctx, `SELECT changes()`)
		if err == nil {
			_ = res
		}
		added++
	}
	return added, nil
}

func (r ProxyRepository) Toggle(ctx context.Context, proxyID int64) (bool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT is_active FROM proxies WHERE id = ?`, proxyID)
	var current bool
	if err := row.Scan(&current); err != nil {
		return false, err
	}
	next := !current
	_, err := r.db.ExecContext(ctx, `UPDATE proxies SET is_active = ? WHERE id = ?`, next, proxyID)
	return next, err
}

func (r ProxyRepository) Delete(ctx context.Context, proxyID int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM proxies WHERE id = ?`, proxyID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r ProxyRepository) CheckAll(ctx context.Context) error {
	items, err := r.List(ctx, domainproxy.ListFilter{})
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 8 * time.Second}
	for _, item := range items {
		proxyURL, err := url.Parse(item.URL)
		if err != nil {
			_, _ = r.db.ExecContext(ctx, `UPDATE proxies SET fail_count = fail_count + 1, last_checked = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339Nano), item.ID)
			continue
		}
		transport := &http.Transport{Proxy: http.ProxyURL(proxyURL)}
		reqClient := &http.Client{Timeout: 8 * time.Second, Transport: transport}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://httpbin.org/ip", nil)
		resp, err := reqClient.Do(req)
		if err == nil && resp.StatusCode == 200 {
			_, _ = r.db.ExecContext(ctx, `UPDATE proxies SET success_count = success_count + 1, last_checked = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339Nano), item.ID)
			_ = resp.Body.Close()
			continue
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		_, _ = r.db.ExecContext(ctx, `UPDATE proxies SET fail_count = fail_count + 1, last_checked = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339Nano), item.ID)
	}
	_ = client
	return nil
}

var _ domainproxy.Repository = ProxyRepository{}
