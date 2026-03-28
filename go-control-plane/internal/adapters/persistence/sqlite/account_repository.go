package sqliteadapter

import (
	"context"
	"database/sql"
	"strings"

	accountquery "go-control-plane/internal/application/query/account"
	domainaccount "go-control-plane/internal/domain/account"
)

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) AccountRepository {
	return AccountRepository{db: db}
}

func (r AccountRepository) GetByID(ctx context.Context, id int64) (domainaccount.Account, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, platform, email, status, cashier_url, trial_end_time, created_at, updated_at
		 FROM accounts WHERE id = ?`,
		id,
	)

	var account domainaccount.Account
	var createdAt string
	var updatedAt string
	if err := row.Scan(
		&account.ID,
		&account.Platform,
		&account.Email,
		&account.Status,
		&account.CashierURL,
		&account.TrialEndTime,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domainaccount.Account{}, err
	}
	account.CreatedAt = parseTime(createdAt)
	account.UpdatedAt = parseTime(updatedAt)
	return account, nil
}

func (r AccountRepository) List(ctx context.Context, filter accountquery.ListAccountsFilter) (int, []domainaccount.Account, error) {
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	clauses := []string{}
	args := []any{}
	if filter.Platform != "" {
		clauses = append(clauses, "platform = ?")
		args = append(args, filter.Platform)
	}
	if filter.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Email != "" {
		clauses = append(clauses, "email LIKE ?")
		args = append(args, "%"+filter.Email+"%")
	}

	whereSQL := ""
	if len(clauses) > 0 {
		whereSQL = " WHERE " + strings.Join(clauses, " AND ")
	}

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM accounts`+whereSQL, args...).Scan(&total); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return 0, []domainaccount.Account{}, nil
		}
		return 0, nil, err
	}

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, pageSize, (page-1)*pageSize)
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, platform, email, status, cashier_url, trial_end_time, created_at, updated_at
		 FROM accounts`+whereSQL+` ORDER BY datetime(created_at) DESC LIMIT ? OFFSET ?`,
		queryArgs...,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return 0, []domainaccount.Account{}, nil
		}
		return 0, nil, err
	}
	defer rows.Close()

	items := make([]domainaccount.Account, 0)
	for rows.Next() {
		var account domainaccount.Account
		var createdAt string
		var updatedAt string
		if err := rows.Scan(&account.ID, &account.Platform, &account.Email, &account.Status, &account.CashierURL, &account.TrialEndTime, &createdAt, &updatedAt); err != nil {
			return 0, nil, err
		}
		account.CreatedAt = parseTime(createdAt)
		account.UpdatedAt = parseTime(updatedAt)
		items = append(items, account)
	}
	return total, items, rows.Err()
}

func (r AccountRepository) GetDashboardStats(ctx context.Context) (accountquery.DashboardStatsResult, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM accounts`).Scan(&total); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return accountquery.DashboardStatsResult{Total: 0, ByPlatform: map[string]int64{}, ByStatus: map[string]int64{}}, nil
		}
		return accountquery.DashboardStatsResult{}, err
	}

	byPlatform := map[string]int64{}
	rows, err := r.db.QueryContext(ctx, `SELECT platform, COUNT(*) FROM accounts GROUP BY platform`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var key string
			var value int64
			if err := rows.Scan(&key, &value); err != nil {
				return accountquery.DashboardStatsResult{}, err
			}
			byPlatform[key] = value
		}
	} else if !strings.Contains(strings.ToLower(err.Error()), "no such table") {
		return accountquery.DashboardStatsResult{}, err
	}

	byStatus := map[string]int64{}
	rows2, err := r.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM accounts GROUP BY status`)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var key string
			var value int64
			if err := rows2.Scan(&key, &value); err != nil {
				return accountquery.DashboardStatsResult{}, err
			}
			byStatus[key] = value
		}
	} else if !strings.Contains(strings.ToLower(err.Error()), "no such table") {
		return accountquery.DashboardStatsResult{}, err
	}

	return accountquery.DashboardStatsResult{Total: total, ByPlatform: byPlatform, ByStatus: byStatus}, nil
}
