package securitycommand

import (
	"crypto/subtle"
	"errors"
	"strings"
)

const InternalCallbackTokenHeader = "X-AAR-Internal-Callback-Token"

var ErrInvalidInternalCallbackToken = errors.New("invalid internal callback token")

func ValidateInternalCallbackToken(expected string, provided string) error {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return nil
	}
	if subtle.ConstantTimeCompare([]byte(expected), []byte(strings.TrimSpace(provided))) != 1 {
		return ErrInvalidInternalCallbackToken
	}
	return nil
}
