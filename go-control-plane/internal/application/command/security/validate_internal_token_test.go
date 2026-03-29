package securitycommand

import "testing"

func TestValidateInternalCallbackToken(t *testing.T) {
	t.Run("allows empty expected token", func(t *testing.T) {
		if err := ValidateInternalCallbackToken("", ""); err != nil {
			t.Fatalf("expected empty token validation to pass, got %v", err)
		}
	})

	t.Run("rejects mismatched token", func(t *testing.T) {
		if err := ValidateInternalCallbackToken("secret", "wrong"); err == nil {
			t.Fatal("expected mismatched token to fail")
		}
	})

	t.Run("accepts matching token", func(t *testing.T) {
		if err := ValidateInternalCallbackToken("secret", "secret"); err != nil {
			t.Fatalf("expected matching token to pass, got %v", err)
		}
	})
}
