package logger

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// New creates a new zerolog logger
func New(level string) zerolog.Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		log.Error().Msgf("cannot parse level %s, using 'info'", level)
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	return logger
}
