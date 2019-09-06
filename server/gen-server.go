package server

import (
	"me_transactions/cfg"
	"me_transactions/service"

	"github.com/halturin/ergonode"
)

type goGenServ struct {
	ergonode.GenServer
	completeChan chan bool
	srv          service.IMeTransactionService
	cfg          *cfg.Config
}

//
// TODO: check this. Seems ergonode does not have interface that satisfy both ergonode.Process and ergonode.GenServerInt
func NewGenServer(srv service.IMeTransactionService, cfg *cfg.Config) (*goGenServ, chan bool) {
	pg2CompleteChan := make(chan bool)
	return &goGenServ{
		completeChan: pg2CompleteChan,
		srv:          srv,
		cfg:          cfg,
	}, pg2CompleteChan
}
