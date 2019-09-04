package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/halturin/ergonode/etf"
	"github.com/rs/zerolog/log"

	"me_transactions/server/entity"
)

// HandleCall serves incoming messages sending via gen_server:call
func (gs *goGenServ) HandleCall(from *etf.Tuple, message *etf.Term, state interface{}) (int, *etf.Term, interface{}) {
	errorReplyTerm := etf.Term(etf.Tuple{etf.Atom("error"), etf.Atom("unknown_request")})
	codeOne := 1
	switch req := (*message).(type) {
	case etf.Atom:
		switch string(req) {
		case "pid":
			rpl := etf.Term(etf.Pid(gs.Self))
			return codeOne, &rpl, state
		}
	case etf.Tuple:
		if len(req) != 4 {
			return codeOne, &errorReplyTerm, state
		}
		args, ok := req[2].(string)
		if !ok {
			return codeOne, &errorReplyTerm, state
		}
		var requestID string
		if str, ok := req[3].(string); ok {
			requestID = str
		}

		logger := log.With().Str("request_id", requestID).Logger()

		var request entity.Request
		if err := json.Unmarshal([]byte(args), &request); err != nil {
			return codeOne, &errorReplyTerm, state
		}
		logAllOps(request, logger)
		if len(request.Operations) == 0 {
			return codeOne, &errorReplyTerm, state
		}
		rsp, err := gs.srv.HandleCall(context.TODO(), &request, logger)
		if err != nil {
			logger.Error().Msg(err.Error())
			rpl := etf.Term(etf.Tuple{etf.Atom("error"), rsp})
			return codeOne, &rpl, state
		}
	default:
		rpl := etf.Term(etf.Tuple{etf.Atom("error"), errorReplyTerm})
		return codeOne, &rpl, state
	}

	rpl := etf.Term(etf.Atom("ok"))
	return codeOne, &rpl, state
}

func logAllOps(req entity.Request, logger zerolog.Logger) {
	var logMessage bytes.Buffer
	logMessage.WriteString("Received message. Operations: [")
	for i, operation := range req.Operations {
		if i != 0 {
			logMessage.WriteString(", ")
		}
		switch operation.Operation {
		case "insert":
			logMessage.WriteString(fmt.Sprintf("{collection: %s, operation: %s}", operation.Collection, operation.Operation))
		case "update_one":
			logMessage.WriteString(fmt.Sprintf("{collection: %s, operation: %s, filter: %s}", operation.Collection, operation.Operation, operation.Filter))
		case "upsert_one":
			logMessage.WriteString(fmt.Sprintf("{collection: %s, operation: %s, filter: %s}", operation.Collection, operation.Operation, operation.Filter))
		case "delete_one":
			logMessage.WriteString(fmt.Sprintf("{collection: %s, operation: %s, filter: %s}", operation.Collection, operation.Operation, operation.Filter))
		}
	}
	logMessage.WriteString("]")
	logger.Warn().Msg(logMessage.String())
}
