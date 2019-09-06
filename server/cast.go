package server

import (
	"os"

	"github.com/halturin/ergonode/etf"
	"github.com/rs/zerolog/log"
)

// HandleCast serves incoming messages sending via gen_server:cast
func (gs *goGenServ) HandleCast(message *etf.Term, state interface{}) (int, interface{}) {
	logger := log.With().Logger()
	switch req := (*message).(type) {
	case etf.Atom:
		switch string(req) {
		case "check":
			logger.Debug().Msgf("Received health check message")
			f, err := os.Create(gs.cfg.HealthCheckPath)
			if err != nil {
				logger.Warn().Msgf("Can't create health check file")
			}
			f.Close()
			return 1, state
		}
	}
	return 1, state
}
