package server

import (
	"github.com/halturin/ergonode/etf"
)

// HandleCast serves incoming messages sending via gen_server:cast
func (gs *goGenServ) HandleCast(message *etf.Term, state interface{}) (int, interface{}) {
	return 0, state
}
