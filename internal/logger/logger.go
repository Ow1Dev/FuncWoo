package logger

import (
	"io"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func InitLog(w io.Writer, debug bool) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(w).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}
