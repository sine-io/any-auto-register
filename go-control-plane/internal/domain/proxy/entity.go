package proxy

import "time"

type Proxy struct {
	ID           int64
	URL          string
	Region       string
	SuccessCount int64
	FailCount    int64
	IsActive     bool
	LastChecked  *time.Time
}
