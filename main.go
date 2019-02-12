package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/halturin/ergonode"
	"github.com/halturin/ergonode/etf"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
)

// GenServer implementation structure
type goGenServ struct {
	ergonode.GenServer
	completeChan chan bool
}

type State struct {
	dbClient *mongo.Client
	dbCtx    context.Context
}

type Operation struct {
	Operation  string `json:"operation"`
	Collection string `json:"collection"`
	Filter     string `json:"filter"`
	Set        string `json:"set"`
}

var (
	mongoURL string
	SrvName  string
	NodeName string
	Cookie   string
	err      error
	EpmdPort int
)

// Init initializes process state using arbitrary arguments
func (gs *goGenServ) Init(args ...interface{}) (state interface{}) {
	// Initialize new instance of goGenServ structure which implements Process behaviour
	var client, err = mongo.NewClient(mongoURL)
	if err != nil {
		var buffer bytes.Buffer
		buffer.WriteString("Failed to connect to mongo: ")
		buffer.WriteString(err.Error())
		panic(buffer.String())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		var buffer bytes.Buffer
		buffer.WriteString("Failed to connect to mongo: ")
		buffer.WriteString(err.Error())
		panic(buffer.String())
	}

	// Self-registration with name go_srv
	gs.Node.Register(etf.Atom(SrvName), gs.Self)

	// Store first argument as channel
	gs.completeChan = args[0].(chan bool)

	return State{dbClient: client, dbCtx: ctx}
}

// HandleCast serves incoming messages sending via gen_server:cast
func (gs *goGenServ) HandleCast(message *etf.Term, state interface{}) (code int, stateout interface{}) {
	return
}

// HandleCall serves incoming messages sending via gen_server:call
func (gs *goGenServ) HandleCall(from *etf.Tuple, message *etf.Term, state interface{}) (code int, reply *etf.Term, stateout interface{}) {
	inState := state.(State)
	stateout = state
	code = 1
	replyTerm := etf.Term(etf.Tuple{etf.Atom("error"), etf.Atom("unknown_request")})
	reply = &replyTerm

	switch req := (*message).(type) {
	case etf.Atom:
		switch string(req) {
		case "pid":
			replyTerm = etf.Term(etf.Pid(gs.Self))
			reply = &replyTerm
		}

	case string:
		var operations []Operation
		json.Unmarshal([]byte(req), &operations)
		client := inState.dbClient
		ctx := inState.dbCtx

		session, err := client.StartSession()
		if err != nil {
			replyTerm := etf.Term("Failed to start session")
			reply = &replyTerm
			return
		}
		session.StartTransaction()
		defer session.EndSession(ctx)
		for _, operation := range operations {
			collection := client.Database("medical_events").Collection(operation.Collection)
			if err != nil {
				replyTerm := etf.Term("Failed to get collection")
				reply = &replyTerm
				return
			}

			switch operation.Operation {
			case "insert":
				data, _ := base64.StdEncoding.DecodeString(operation.Set)

				var f interface{}
				bson.Unmarshal(data, &f)
				a, err := collection.InsertOne(context.Background(), data)
				if err != nil {
					session.AbortTransaction(ctx)
					var buffer bytes.Buffer
					buffer.WriteString("Aborting transaction. ")
					buffer.WriteString(err.Error())
					replyTerm := etf.Term(etf.Tuple{etf.Atom("error"), buffer.String()})
					reply = &replyTerm
					return
				}
				fmt.Printf("%s\n", a)
			case "update_one":
				filter, _ := base64.StdEncoding.DecodeString(operation.Filter)
				set, _ := base64.StdEncoding.DecodeString(operation.Set)

				var f interface{}
				bson.Unmarshal(filter, &f)
				bson.Unmarshal(set, &f)
				a, err := collection.UpdateOne(context.Background(), filter, set)
				if err != nil {
					session.AbortTransaction(ctx)
					var buffer bytes.Buffer
					buffer.WriteString("Aborting transaction. ")
					buffer.WriteString(err.Error())
					replyTerm := etf.Term(etf.Tuple{etf.Atom("error"), buffer.String()})
					reply = &replyTerm
					return
				}
				fmt.Printf("Matched: %d, Modified: %d\n", a.MatchedCount, a.ModifiedCount)
			}
		}

		session.CommitTransaction(ctx)
		result := etf.Term(etf.Atom("ok"))
		reply = &result
	}
	return
}

// HandleInfo serves all another incoming messages (Pid ! message)
func (gs *goGenServ) HandleInfo(message *etf.Term, state interface{}) (code int, stateout interface{}) {
	fmt.Printf("HandleInfo: %#v\n", *message)
	stateout = state
	code = 0
	return
}

// Terminate called when process died
func (gs *goGenServ) Terminate(reason int, state interface{}) {
	fmt.Printf("Terminate: %#v\n", reason)
}

func init() {
	flag.StringVar(&mongoURL, "mongo_url", "mongodb://localhost:27017/medical_events?replicaSet=replicaTest", "mongo connect url")
	flag.StringVar(&SrvName, "gen_server", "mongo_transaction", "gen_server name")
	flag.StringVar(&NodeName, "name", "examplenode@127.0.0.1", "node name")
	flag.StringVar(&Cookie, "cookie", "123", "cookie for interaction with erlang cluster")
	flag.IntVar(&EpmdPort, "epmd_port", 15151, "epmd port")
}

func main() {
	flag.Parse()

	// Initialize new node with given name and cookie
	n := ergonode.Create(NodeName, uint16(EpmdPort), Cookie)

	// Create channel to receive message when main process should be stopped
	completeChan := make(chan bool)

	gs := new(goGenServ)
	// Spawn process with one arguments
	n.Spawn(gs, completeChan)
	fmt.Println("Started node")

	// Wait to stop
	<-completeChan

	return
}

// session = db.getMongo().startSession( { readPreference: { mode: "primary" } } );
// employeesCollection = session.getDatabase("test").employees;
// session.startTransaction( { readConcern: { level: "snapshot" }, writeConcern: { w: "majority" } } );
// employeesCollection.insertOne( { name: "test" }  );
// session.commitTransaction();
