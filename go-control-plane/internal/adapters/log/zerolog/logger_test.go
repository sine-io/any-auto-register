package zerologadapter

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewWritesJSONLogs(t *testing.T) {
	var buf bytes.Buffer
	logger := NewWithWriter("info", &buf)
	logger.Info().Msg("boot")

	output := buf.String()
	if !strings.Contains(output, "\"level\":\"info\"") {
		t.Fatalf("expected info level in output, got %s", output)
	}
	if !strings.Contains(output, "\"message\":\"boot\"") {
		t.Fatalf("expected message in output, got %s", output)
	}
}
