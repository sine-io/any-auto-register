package task

import "context"

type ListFilter struct {
	Page     int
	PageSize int
}

type Repository interface {
	List(ctx context.Context, filter ListFilter) (int, []TaskRun, error)
}
