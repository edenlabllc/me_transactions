package server

import (
	"github.com/halturin/ergonode/etf"
)

// Init initializes process state using arbitrary arguments
func (gs *goGenServ) Init(args ...interface{}) (state interface{}) {
	// Self-registration with name SrvName
	gs.Node.Register(etf.Atom("examplenode@127.0.0.1"), gs.Self)
	return nil
}
