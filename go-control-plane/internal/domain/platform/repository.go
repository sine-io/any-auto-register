package platform

import "context"

type Repository interface {
	List(ctx context.Context) ([]Platform, error)
}
