package proxy

import "context"

type ListFilter struct{}

type Repository interface {
	List(ctx context.Context, filter ListFilter) ([]Proxy, error)
}
