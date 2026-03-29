package sqliteadapter

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	accountcommand "go-control-plane/internal/application/command/account"
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
		`SELECT id, platform, email, password, region, token, status, cashier_url, trial_end_time, created_at, updated_at
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
		&account.Password,
		&account.Region,
		&account.Token,
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
		`SELECT id, platform, email, password, region, token, status, cashier_url, trial_end_time, created_at, updated_at
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
		if err := rows.Scan(&account.ID, &account.Platform, &account.Email, &account.Password, &account.Region, &account.Token, &account.Status, &account.CashierURL, &account.TrialEndTime, &createdAt, &updatedAt); err != nil {
			return 0, nil, err
		}
		account.CreatedAt = parseTime(createdAt)
		account.UpdatedAt = parseTime(updatedAt)
		items = append(items, account)
	}
	return total, items, rows.Err()
}

func (r AccountRepository) CreateAccount(ctx context.Context, cmd accountcommand.CreateAccountCommand) (domainaccount.Account, error) {
	status := cmd.Status
	if status == "" {
		status = "registered"
	}
	now := time.Now().UTC()
	result, err := r.db.ExecContext(
		ctx,
		`INSERT INTO accounts (
			platform, email, password, user_id, region, token, status, trial_end_time, cashier_url, extra_json, created_at, updated_at
		) VALUES (?, ?, ?, '', '', ?, ?, ?, ?, '{}', ?, ?)`,
		cmd.Platform,
		cmd.Email,
		cmd.Password,
		cmd.Token,
		status,
		cmd.TrialEndTime,
		cmd.CashierURL,
		now,
		now,
	)
	if err != nil {
		return domainaccount.Account{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domainaccount.Account{}, err
	}
	return r.GetByID(ctx, id)
}

func (r AccountRepository) UpdateAccount(ctx context.Context, cmd accountcommand.UpdateAccountCommand) (domainaccount.Account, error) {
	account, err := r.GetByID(ctx, cmd.AccountID)
	if err != nil {
		return domainaccount.Account{}, err
	}
	if cmd.Status != nil {
		account.Status = *cmd.Status
	}
	if cmd.Token != nil {
		account.Token = *cmd.Token
	}
	if cmd.TrialEndTime != nil {
		account.TrialEndTime = *cmd.TrialEndTime
	}
	if cmd.CashierURL != nil {
		account.CashierURL = *cmd.CashierURL
	}
	account.UpdatedAt = time.Now().UTC()

	_, err = r.db.ExecContext(
		ctx,
		`UPDATE accounts
		 SET status = ?, token = ?, trial_end_time = ?, cashier_url = ?, updated_at = ?
		 WHERE id = ?`,
		account.Status,
		account.Token,
		account.TrialEndTime,
		account.CashierURL,
		account.UpdatedAt,
		cmd.AccountID,
	)
	if err != nil {
		return domainaccount.Account{}, err
	}
	return r.GetByID(ctx, cmd.AccountID)
}

func (r AccountRepository) DeleteAccount(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return err
}

func (r AccountRepository) BatchDeleteAccounts(ctx context.Context, ids []int64) (int, []int64, error) {
	deleted := 0
	notFound := make([]int64, 0)
	for _, id := range ids {
		result, err := r.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = ?`, id)
		if err != nil {
			return 0, nil, err
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return 0, nil, err
		}
		if rowsAffected == 0 {
			notFound = append(notFound, id)
			continue
		}
		deleted++
	}
	return deleted, notFound, nil
}

func (r AccountRepository) ImportAccounts(ctx context.Context, platform string, lines []string) (int, error) {
	created := 0
	now := time.Now().UTC()
	for _, line := range lines {
		parts := strings.Fields(strings.TrimSpace(line))
		if len(parts) < 2 {
			continue
		}
		email := parts[0]
		password := parts[1]
		cashierURL := ""
		if len(parts) > 2 {
			cashierURL = parts[2]
		}
		if _, err := r.db.ExecContext(
			ctx,
			`INSERT INTO accounts (
				platform, email, password, user_id, region, token, status, trial_end_time, cashier_url, extra_json, created_at, updated_at
			) VALUES (?, ?, ?, '', '', '', 'registered', 0, ?, '{}', ?, ?)`,
			platform,
			email,
			password,
			cashierURL,
			now,
			now,
		); err != nil {
			return 0, fmt.Errorf("import account %s: %w", email, err)
		}
		created++
	}
	return created, nil
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
