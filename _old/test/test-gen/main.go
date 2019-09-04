package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/halturin/ergonode"
	"github.com/halturin/ergonode/etf"
	"github.com/rs/zerolog/log"
)

var (
	ReceiverNodeName string
	ReceiverCookie   string

	TransmitterNodeName string
	TransmitterCookie   string
)

type transServ struct {
	ergonode.GenServer
	transmitterChan chan bool
}

func init() {
	ReceiverNodeName = "examplenode@127.0.0.1 5 1 0"
	ReceiverCookie = "123"

	TransmitterNodeName = "transmitter@127.0.0.1"
	TransmitterCookie = "123"

}

func (pg2 *transServ) Init(args ...interface{}) (state interface{}) {
	pg2.Node.Register(etf.Atom("pg2"), pg2.Self)
	return nil
}

func (pg2 *transServ) HandleCall(from *etf.Tuple, message *etf.Term, state interface{}) (code int, reply *etf.Term, stateout interface{}) {
	stateout = state
	code = 1
	replyTerm := etf.Term(etf.Atom("ok"))
	reply = &replyTerm
	return
}

func (pg2 *transServ) HandleCast(message *etf.Term, state interface{}) (code int, stateout interface{}) {
	return
}

// HandleInfo serves all another incoming messages (Pid ! message)
func (pg2 *transServ) HandleInfo(message *etf.Term, state interface{}) (code int, stateout interface{}) {
	fmt.Printf("HandleInfo: %#v\n", *message)
	stateout = state
	code = 0
	return
}

// Terminate called when process died
func (pg2 *transServ) Terminate(reason int, state interface{}) {
	fmt.Printf("Terminate: %#v\n", reason)
}

func main() {
	n := ergonode.Create(TransmitterNodeName, TransmitterCookie)
	log.Info().Msg("Started erlang transmitter")
	transmitterChan := make(chan bool)
	tGS := new(transServ)
	n.Spawn(tGS, transmitterChan)
	log.Info().Msg("Started erlang transmitter gen server")

	generateAndTransmitMsg(tGS)
	return
}

func generateAndTransmitMsg(gs *transServ) (*etf.Term, error) {
	set, _ := bson.Marshal(Set{
		ID:    "13",
		Block: "321",
	})
	r := base64.StdEncoding.EncodeToString(set)
	op := Operation{
		Operation:  "insert",
		Collection: "lol",
		Filter:     "",
		Set:        r,
		Id:         "123",
	}
	request := Request{
		ActorID:    "123",
		PatientID:  "321",
		Operations: []Operation{op},
	}
	byteReq, err := json.Marshal(request)
	message := etf.Term(etf.Tuple{1, 2, byteReq, "reqID123"})

	//to := etf.Tuple{etf.Atom(ReceiverCookie), etf.Atom(ReceiverNodeName)}
	toPID := etf.Pid{etf.Atom("examplenode@127.0.0.1"), 4, 1, 0}

	answer, err := gs.Call(toPID, &message)
	if err != nil {
		log.Warn().Msgf("Error on transmit %+v", err)
		return nil, err
	}
	a, ok := etf.StringTerm(answer)
	if !ok {
		log.Info().Msgf("Answer %v", answer)
	}
	log.Info().Msgf("Answer %s", a)
	return answer, nil
}

type Operation struct {
	Operation  string `json:"operation"`
	Collection string `json:"collection"`
	Filter     string `json:"filter"`
	Set        string `json:"set"`
	Id         string `json:"id"`
}

type Request struct {
	ActorID    string      `json:"actor_id"`
	PatientID  string      `json:"patient_id"`
	Operations []Operation `json:"operations"`
}

type Set struct {
	ID    string `bson:"id"`
	Block string `bson:"block"`
}
