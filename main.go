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
	"time"

	"github.com/halturin/ergonode"
	"github.com/halturin/ergonode/etf"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// GenServer implementation structure
type goGenServ struct {
	ergonode.GenServer
	completeChan chan bool
}

type pg2Serv struct {
	ergonode.GenServer
	pg2CompleteChan chan bool
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
type Request struct {
	ActorID    string      `json:"actor_id"`
	PatientID  string      `json:"patient_id"`
	Operations []Operation `json:"operations"`
}

var (
	mongoURL               string
	healthCheckPath        string
	SrvName                string
	DBPoolSize             uint64
	DBWriteConcern         int
	AuditLogCollectionName string
	AuditLogEnabled        string
	NodeName               string
	Cookie                 string
	LogLevel               string
	err                    error
	EpmdPort               int
)

func (pg2 *pg2Serv) Init(args ...interface{}) (state interface{}) {
	pg2.Node.Register(etf.Atom("pg2"), pg2.Self)
	return nil
}

func (pg2 *pg2Serv) HandleCall(from *etf.Tuple, message *etf.Term, state interface{}) (code int, reply *etf.Term, stateout interface{}) {
	stateout = state
	code = 1
	replyTerm := etf.Term(etf.Atom("ok"))
	reply = &replyTerm
	return
}

func (pg2 *pg2Serv) HandleCast(message *etf.Term, state interface{}) (code int, stateout interface{}) {
	return
}

// HandleInfo serves all another incoming messages (Pid ! message)
func (pg2 *pg2Serv) HandleInfo(message *etf.Term, state interface{}) (code int, stateout interface{}) {
	fmt.Printf("HandleInfo: %#v\n", *message)
	stateout = state
	code = 0
	return
}

// Terminate called when process died
func (pg2 *pg2Serv) Terminate(reason int, state interface{}) {
	fmt.Printf("Terminate: %#v\n", reason)
}

// Init initializes process state using arbitrary arguments
func (gs *goGenServ) Init(args ...interface{}) (state interface{}) {
	// Initialize new instance of goGenServ structure which implements Process behaviour
	var client, err = mongo.NewClient(options.Client().ApplyURI(mongoURL).SetMaxPoolSize(DBPoolSize))
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
	logger := log.With().Logger()
	stateout = state
	switch req := (*message).(type) {
	case etf.Atom:
		switch string(req) {
		case "check":
			logger.Debug().Msgf("Received health check message")
			f, err := os.Create(healthCheckPath)
			if err != nil {
				logger.Warn().Msgf("Can't create health check file")
			}
			f.Close()
			return
		}
	}
	return
}

// HandleCall serves incoming messages sending via gen_server:call
func (gs *goGenServ) HandleCall(from *etf.Tuple, message *etf.Term, state interface{}) (code int, reply *etf.Term, stateout interface{}) {
	inState := state.(State)
	stateout = state
	code = 0
	fromPid := (*from)[0].(etf.Pid)
	fromRef := (*from)[1]

	go func() {
		replyTerm := etf.Term(etf.Tuple{etf.Atom("error"), etf.Atom("unknown_request")})
		switch req := (*message).(type) {
		case etf.Atom:
			switch string(req) {
			case "pid":
				replyTerm = etf.Term(etf.Pid(gs.Self))
			}

		case etf.Tuple:
			if len(req) == 4 {
				args := req[2].(string)
				requestID := ""

				if str, ok := req[3].(string); ok {
					requestID = str
				}

				logger := log.With().Str("request_id", requestID).Logger()

				var request Request
				json.Unmarshal([]byte(args), &request)

				var logMessage bytes.Buffer
				logMessage.WriteString("Received message. Operations: [")
				for i, operation := range request.Operations {
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

				if len(request.Operations) == 0 {
					replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), "No valid operations"})
					return
				}
				client := inState.dbClient
				ctx := inState.dbCtx

				session, err := client.StartSession()
				if err != nil {
					logger.Warn().Msgf("Failed to start session: %s", err)
					replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), "Failed to start session"})
					sendResponse(gs, fromPid, fromRef, replyTerm)
					return
				}
				logger.Debug().Msgf("Started session")

				database := client.Database("medical_data")
				concern := writeconcern.WMajority()
				if DBWriteConcern != 0 {
					concern = writeconcern.W(DBWriteConcern)
				}

				transactionFn := func(sctx mongo.SessionContext) error {
					// Start a transaction in a session
					err := sctx.StartTransaction(options.Transaction().
						SetWriteConcern(writeconcern.New(concern)),
					)
					if err != nil {
						logger.Error().Msgf("Failed to start transaction: %s", err)
						replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), err.Error()})
						return err
					}
					logger.Debug().Msgf("Started transaction")
					auditLogCollection := database.Collection(AuditLogCollectionName)

					for _, operation := range request.Operations {
						collection := database.Collection(operation.Collection)
						logger.Debug().Msgf("Processing %s in %s collection", operation.Operation, operation.Collection)

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

							var d interface{}
							bson.Unmarshal(data, &d)
							a, err := collection.InsertOne(sctx, d)
							if err != nil {
								logger.Warn().Msgf("Aborting transaction: %s", err.Error())
								logger.Warn().Msgf("Failed args: %s", args)
								sctx.AbortTransaction(sctx)
								var buffer bytes.Buffer
								buffer.WriteString("Aborting transaction. ")
								buffer.WriteString(err.Error())
								replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), buffer.String()})
								return err
							} else if AuditLogEnabled == "true" {
								saveInsertAuditLog(sctx, auditLogCollection, operation, d, request.ActorID, request.PatientID, logger)
							}
							logger.Info().Msgf("Inserted: %s", a)
						case "update_one":
							filter, err := base64.StdEncoding.DecodeString(operation.Filter)
							if err != nil {
								logger.Warn().Msgf("Invalid base64 string on filter update: %s", operation.Filter)
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
							var s interface{}
							bson.Unmarshal(set, &s)
							a, err := collection.UpdateOne(sctx, f, s)
							if err != nil {
								logger.Warn().Msgf("Aborting transaction. %s", err.Error())
								logger.Warn().Msgf("Failed args: %s", args)
								sctx.AbortTransaction(sctx)
								var buffer bytes.Buffer
								buffer.WriteString("Aborting transaction. ")
								buffer.WriteString(err.Error())
								replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), buffer.String()})
								return err
							} else if AuditLogEnabled == "true" {
								saveUpdateAuditLog(sctx, auditLogCollection, operation, f, s, request.ActorID, request.PatientID, a, logger)
							}
							logger.Info().Msgf("Matched: %d, Modified: %d", a.MatchedCount, a.ModifiedCount)
						case "upsert_one":
							filter, err := base64.StdEncoding.DecodeString(operation.Filter)
							if err != nil {
								logger.Warn().Msgf("Invalid base64 string on filter upsert: %s", operation.Filter)
								var buffer bytes.Buffer
								buffer.WriteString("Invalid base64 string. ")
								buffer.WriteString(err.Error())
								replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), buffer.String()})
								return err
							}
							set, err := base64.StdEncoding.DecodeString(operation.Set)
							if err != nil {
								logger.Warn().Msgf("Invalid base64 string on set upsert: %s", operation.Set)
								var buffer bytes.Buffer
								buffer.WriteString("Invalid base64 string. ")
								buffer.WriteString(err.Error())
								replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), buffer.String()})
								return err
							}

							var f interface{}
							bson.Unmarshal(filter, &f)
							var s interface{}
							bson.Unmarshal(set, &s)
							var upsert = true
							var upsertOptions = options.UpdateOptions{Upsert: &upsert}
							a, err := collection.UpdateOne(sctx, f, s, &upsertOptions)
							if err != nil {
								logger.Warn().Msgf("Aborting transaction. %s", err.Error())
								logger.Warn().Msgf("Failed args: %s", args)
								sctx.AbortTransaction(sctx)
								var buffer bytes.Buffer
								buffer.WriteString("Aborting transaction. ")
								buffer.WriteString(err.Error())
								replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), buffer.String()})
								return err
							} else if AuditLogEnabled == "true" {
								saveUpdateAuditLog(sctx, auditLogCollection, operation, f, s, request.ActorID, request.PatientID, a, logger)
							}
							logger.Info().Msgf("Matched: %d, Modified: %d, Upserted: %d", a.MatchedCount, a.ModifiedCount, a.UpsertedCount)
						case "delete_one":
							filter, err := base64.StdEncoding.DecodeString(operation.Filter)
							if err != nil {
								logger.Warn().Msgf("Invalid base64 string on delete: %s", operation.Filter)
								var buffer bytes.Buffer
								buffer.WriteString("Invalid base64 string. ")
								buffer.WriteString(err.Error())
								replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), buffer.String()})
								return err
							}

							var f interface{}
							bson.Unmarshal(filter, &f)
							a, err := collection.DeleteOne(sctx, f)
							if err != nil {
								logger.Warn().Msgf("Aborting transaction: %s", err.Error())
								logger.Warn().Msgf("Failed args: %s", args)
								sctx.AbortTransaction(sctx)
								var buffer bytes.Buffer
								buffer.WriteString("Aborting transaction. ")
								buffer.WriteString(err.Error())
								replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), buffer.String()})
								return err
							} else if AuditLogEnabled == "true" {
								saveDeleteAuditLog(sctx, auditLogCollection, operation, f, request.ActorID, logger)
							}
							logger.Info().Msgf("Deleted: %d", a.DeletedCount)
						default:
							logger.Info().Msgf("Invalid operation")
						}
					}

					// Committing transaction
					for {
						err = sctx.CommitTransaction(sctx)
						switch e := err.(type) {
						case nil:
							return nil
						case mongo.CommandError:
							if e.HasErrorLabel("UnknownTransactionCommitResult") {
								continue
							}
							logger.Info().Msgf("Retry transaction: %s", err)
							replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), err.Error()})
							return err
						default:
							replyTerm = etf.Term(etf.Tuple{etf.Atom("error"), err.Error()})
							return err
						}
					}
				}

				err = mongo.WithSession(ctx, session, func(sctx mongo.SessionContext) error {
					return runTransactionWithRetry(sctx, transactionFn)
				})

				session.EndSession(ctx)
				if err == nil {
					replyTerm = etf.Term(etf.Atom("ok"))
				}
			}
		}

		sendResponse(gs, fromPid, fromRef, replyTerm)
		return
	}()

	return
}

func sendResponse(gs *goGenServ, fromPid etf.Pid, fromRef etf.Term, replyTerm etf.Term) {
	rep := etf.Term(etf.Tuple{fromRef, replyTerm})
	gs.Send(fromPid, &rep)
	return
}

func runTransactionWithRetry(sctx mongo.SessionContext, txnFn func(mongo.SessionContext) error) error {
	for {
		err := txnFn(sctx) // Performs transaction.
		if err == nil {
			return nil
		}

		// If transient error, retry the whole transaction
		if cmdErr, ok := err.(mongo.CommandError); ok && cmdErr.HasErrorLabel("TransientTransactionError") {
			continue
		}
		return err
	}
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

func saveInsertAuditLog(
	sctx mongo.SessionContext,
	auditLogCollection *mongo.Collection,
	operation Operation,
	set interface{},
	actorID string,
	patientID string,
	logger zerolog.Logger) {

	_, err := auditLogCollection.InsertOne(sctx, bson.D{
		{"collection", operation.Collection},
		{"actor_id", actorID},
		{"patient_id", patientID},
		{"params", set},
		{"type", "INSERT"},
		{"inserted_at", time.Now()},
	})
	if err != nil {
		logger.Warn().Msgf("Failed to insert audit log %s", err.Error())
	}
}

func saveUpdateAuditLog(
	sctx mongo.SessionContext,
	auditLogCollection *mongo.Collection,
	operation Operation,
	filter interface{},
	set interface{},
	actorID string,
	patientID string,
	updateResult *mongo.UpdateResult,
	logger zerolog.Logger) {
	var operationType string
	if updateResult.ModifiedCount > 0 {
		operationType = "UPDATE"
	} else if updateResult.UpsertedCount > 0 {
		operationType = "INSERT"
	}

	if operationType != "" {
		_, err := auditLogCollection.InsertOne(sctx, bson.D{
			{"collection", operation.Collection},
			{"actor_id", actorID},
			{"patient_id", patientID},
			{"params", set},
			{"filter", filter},
			{"type", operationType},
			{"inserted_at", time.Now()},
		})
		if err != nil {
			logger.Warn().Msgf("Failed to insert audit log %s", err.Error())
		}
	}
}

func saveDeleteAuditLog(
	sctx mongo.SessionContext,
	auditLogCollection *mongo.Collection,
	operation Operation,
	filter interface{},
	actorID string,
	logger zerolog.Logger) {
	_, err := auditLogCollection.InsertOne(sctx, bson.D{
		{"collection", operation.Collection},
		{"actor_id", actorID},
		{"filter", filter},
		{"type", "DELETE"},
		{"inserted_at", time.Now()},
	})
	if err != nil {
		logger.Warn().Msgf("Failed to insert audit log %s", err.Error())
	}
}

func init() {
	mongoURL = os.Getenv("MONGO_URL")
	if mongoURL == "" {
		flag.StringVar(&mongoURL, "mongo_url", "mongodb://localhost:27017/medical_events?replicaSet=replicaTest", "mongo connect url")
	}

	healthCheckPath = os.Getenv("HEALTH_CHECK_PATH")
	if healthCheckPath == "" {
		flag.StringVar(&healthCheckPath, "health_check", "/tmp/healthy", "health check path")
	}

	dbPoolSize := os.Getenv("DB_POOL_SIZE")
	if dbPoolSize != "" {
		var base = 10
		var size = 64
		a, err := strconv.ParseUint(dbPoolSize, base, size)
		if err != nil {
			panic("Invalid pool size")
		}
		DBPoolSize = uint64(a)
	} else {
		DBPoolSize = 50
	}

	writeConcern := os.Getenv("DB_WRITE_CONCERN")
	if writeConcern != "" {
		var base = 10
		var size = 16
		a, err := strconv.ParseUint(dbPoolSize, base, size)
		if err != nil {
			panic("Invalid write concern")
		}
		DBWriteConcern = int(a)
	}

	SrvName = os.Getenv("GEN_SERVER_NAME")
	if SrvName == "" {
		flag.StringVar(&SrvName, "gen_server", "mongo_transaction", "gen_server name")
	}

	NodeName = os.Getenv("NODE_NAME")
	if NodeName == "" {
		flag.StringVar(&NodeName, "name", "examplenode@127.0.0.1", "node name")
	}

	AuditLogCollectionName = os.Getenv("AUDIT_LOG_COLLECTION")
	if AuditLogCollectionName == "" {
		flag.StringVar(&AuditLogCollectionName, "audit_log_collection", "audit_log", "audit log collection name")
	}

	AuditLogEnabled = os.Getenv("AUDIT_LOG_ENABLED")
	if AuditLogEnabled == "" {
		flag.StringVar(&AuditLogEnabled, "audit_log_enabled", "true", "audit log enabled")
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
	zerolog.TimeFieldFormat = "2006-01-02T15:04:05.999Z"

	log.Logger = log.With().Logger()

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
	n := ergonode.Create(NodeName, Cookie)
	log.Info().Msg("Started erlang node")

	pg2CompleteChan := make(chan bool)

	pg2 := new(pg2Serv)
	// Spawn process with one arguments
	n.Spawn(pg2, pg2CompleteChan)
	log.Info().Msg("Spawned pg2 gen server process")

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
