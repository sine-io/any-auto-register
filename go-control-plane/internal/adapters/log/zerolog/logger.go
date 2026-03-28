package zerologadapter

import (
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

func New(level string) zerolog.Logger {
	return NewWithWriter(level, os.Stdout)
}

func NewWithWriter(level string, writer io.Writer) zerolog.Logger {
	logger := zerolog.New(writer).With().Timestamp().Logger()

	parsedLevel, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		parsedLevel = zerolog.InfoLevel
	}

	return logger.Level(parsedLevel)
}
