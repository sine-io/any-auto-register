package account

import "time"

type Account struct {
	ID           int64
	Platform     string
	Email        string
	Status       string
	CashierURL   string
	TrialEndTime int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
