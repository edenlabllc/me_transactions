package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/halturin/ergonode"
	"github.com/halturin/ergonode/etf"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
	LogLevel string
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
	ctx := context.Background()
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
			return
		}

	case etf.Tuple:
		if len(req) == 4 {
			args := req[2].(string)
			requestID := req[3].(string)
			logger := log.With().Str("request_id", requestID).Logger()

			var operations []Operation
			json.Unmarshal([]byte(args), &operations)

			var logMessage bytes.Buffer
			logMessage.WriteString("Received message. Operations: [")
			for i, operation := range operations {
				if i != 0 {
					logMessage.WriteString(", ")
				}
				switch operation.Operation {
				case "insert":
					logMessage.WriteString(fmt.Sprintf("{collection: %s, operation: %s}", operation.Collection, operation.Operation))
				case "update_one":
					logMessage.WriteString(fmt.Sprintf("{collection: %s, operation: %s, filter: %s}", operation.Collection, operation.Operation, operation.Filter))
				}
			}
			logMessage.WriteString("]")
			logger.Warn().Msg(logMessage.String())

			if len(operations) == 0 {
				replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), "No valid operations"})
				return
			}
			client := inState.dbClient
			ctx := inState.dbCtx

			session, err := client.StartSession()
			if err != nil {
				logger.Warn().Msgf("Failed to start session: %s", err)
				replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), "Failed to start session"})
				return
			}
			logger.Debug().Msgf("Started session")

			database := client.Database("medical_data")

			err = mongo.WithSession(ctx, session, func(sctx mongo.SessionContext) error {
				// Start a transaction in a session
				sctx.StartTransaction()
				logger.Debug().Msgf("Started transaction")

				for _, operation := range operations {
					collection := database.Collection(operation.Collection)
					logger.Info().Msgf("Processing %s in %s collection", operation.Operation, operation.Collection)

					switch operation.Operation {
					case "insert":
						data, err := base64.StdEncoding.DecodeString(operation.Set)
						if err != nil {
							logger.Warn().Msgf("Invalid base64 string on insert: %s", operation.Set)
							var buffer bytes.Buffer
							buffer.WriteString("Invalid base64 string. ")
							buffer.WriteString(err.Error())
							replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), buffer.String()})
							return err
						}

						var f interface{}
						bson.Unmarshal(data, &f)
						a, err := collection.InsertOne(sctx, data)
						if err != nil {
							logger.Warn().Msgf("Aborting transaction: %s", err.Error())
							logger.Warn().Msgf("Failed args: %s", args)
							session.AbortTransaction(sctx)
							var buffer bytes.Buffer
							buffer.WriteString("Aborting transaction. ")
							buffer.WriteString(err.Error())
							replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), buffer.String()})
							return err
						}
						logger.Info().Msgf("Inserted: %s", a)
					case "update_one":
						filter, err := base64.StdEncoding.DecodeString(operation.Filter)
						if err != nil {
							logger.Warn().Msgf("Invalid base64 string on filter update: %s", operation.Set)
							var buffer bytes.Buffer
							buffer.WriteString("Invalid base64 string. ")
							buffer.WriteString(err.Error())
							replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), buffer.String()})
							return err
						}
						set, err := base64.StdEncoding.DecodeString(operation.Set)
						if err != nil {
							logger.Warn().Msgf("Invalid base64 string on set update: %s", operation.Set)
							var buffer bytes.Buffer
							buffer.WriteString("Invalid base64 string. ")
							buffer.WriteString(err.Error())
							replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), buffer.String()})
							return err
						}

						var f interface{}
						bson.Unmarshal(filter, &f)
						bson.Unmarshal(set, &f)
						a, err := collection.UpdateOne(sctx, filter, set)
						if err != nil {
							logger.Warn().Msgf("Aborting transaction. %s", err.Error())
							logger.Warn().Msgf("Failed args: %s", args)
							session.AbortTransaction(sctx)
							var buffer bytes.Buffer
							buffer.WriteString("Aborting transaction. ")
							buffer.WriteString(err.Error())
							replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), buffer.String()})
							return err
						}
						logger.Info().Msgf("Matched: %d, Modified: %d", a.MatchedCount, a.ModifiedCount)
					}
				}

				// Committing transaction
				session.CommitTransaction(sctx)
				return nil
			})
			session.EndSession(ctx)
			if err == nil {
				replyTerm = etf.Term(etf.Atom("ok"))
			}
		}
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
	mongoURL = os.Getenv("MONGO_URL")
	if mongoURL == "" {
		flag.StringVar(&mongoURL, "mongo_url", "mongodb://localhost:27017/medical_events?replicaSet=replicaTest", "mongo connect url")
	}

	SrvName = os.Getenv("GEN_SERVER_NAME")
	if SrvName == "" {
		flag.StringVar(&SrvName, "gen_server", "mongo_transaction", "gen_server name")
	}

	NodeName = os.Getenv("NODE_NAME")
	if NodeName == "" {
		flag.StringVar(&NodeName, "name", "examplenode@127.0.0.1", "node name")
	}

	Cookie = os.Getenv("ERLANG_COOKIE")
	if Cookie == "" {
		flag.StringVar(&Cookie, "cookie", "123", "cookie for interaction with erlang cluster")
	}

	port := os.Getenv("EMPD_PORT")
	if port == "" {
		flag.IntVar(&EpmdPort, "epmd_port", 15151, "epmd port")
	} else {
		EpmdPort, err = strconv.Atoi(port)
		if err != nil {
			panic("Invalid empd port")
		}
	}

	LogLevel := os.Getenv("LOG_LEVEL")
	if LogLevel == "" {
		flag.StringVar(&LogLevel, "log_level", "info", "log level")
	}
}

func main() {
	zerolog.LevelFieldName = "severity"
	zerolog.MessageFieldName = "log"
	zerolog.CallerFieldName = "sourceLocation"
	log.Logger = log.With().Caller().Logger()

	switch LogLevel {
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	flag.Parse()

	// Initialize new node with given name and cookie
	n := ergonode.Create(NodeName, uint16(EpmdPort), Cookie)
	log.Info().Msg("Started erlang node")

	// Create channel to receive message when main process should be stopped
	completeChan := make(chan bool)

	gs := new(goGenServ)
	// Spawn process with one arguments
	n.Spawn(gs, completeChan)
	log.Info().Msg("Spawned gen server process")

	// Wait to stop
	<-completeChan

	return
}

// session = db.getMongo().startSession( { readPreference: { mode: "primary" } } );
// employeesCollection = session.getDatabase("test").employees;
// session.startTransaction( { readConcern: { level: "snapshot" }, writeConcern: { w: "majority" } } );
// employeesCollection.insertOne( { name: "test" }  );
// session.commitTransaction();
