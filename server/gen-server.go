package server

import (
	"me_transactions/service"

	"github.com/halturin/ergonode"
)

type goGenServ struct {
	ergonode.GenServer
	completeChan chan bool
	srv          service.IMeTransactionService
}

//
// TODO: check this. Seems ergonode does not have interface that satisfy both ergonode.Process and ergonode.GenServerInt
func NewGenServer(srv service.IMeTransactionService) (*goGenServ, chan bool) {
	pg2CompleteChan := make(chan bool)
	return &goGenServ{
		completeChan: pg2CompleteChan,
		srv:          srv,
	}, pg2CompleteChan
}
