package server

import "fmt"

// Terminate called when process died
func (gs *goGenServ) Terminate(reason int, state interface{}) {
	fmt.Printf("Terminate: %#v\n", reason)
}
