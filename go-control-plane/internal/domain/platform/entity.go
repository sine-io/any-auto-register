package platform

type Platform struct {
	Name               string
	DisplayName        string
	Version            string
	SupportedExecutors []string
	Available          bool
	AvailabilityReason string
}
