package account

import "context"

type Repository interface {
	GetByID(ctx context.Context, id int64) (Account, error)
}
