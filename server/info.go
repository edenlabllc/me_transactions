package server

import (
	"fmt"

	"github.com/halturin/ergonode/etf"
)

// HandleInfo serves all another incoming messages (Pid ! message)
func (gs *goGenServ) HandleInfo(message *etf.Term, state interface{}) (code int, stateout interface{}) {
	fmt.Printf("HandleInfo: %#v\n", *message)
	stateout = state
	code = 0
	return
}
