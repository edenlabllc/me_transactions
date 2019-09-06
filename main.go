package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/halturin/ergonode"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
  
	"me_transactions/cfg"
	"me_transactions/repo/mongodb"
	"me_transactions/server"
	"me_transactions/service"
)

func main() {
	flag.Parse()
	c, err := cfg.ConfigFromEnv()
	if err != nil {
		panic(err)
	}

	zerolog.LevelFieldName = "severity"
	zerolog.MessageFieldName = "log"
	zerolog.TimeFieldFormat = "2006-01-02T15:04:05.999Z"

	switch c.LogLevel {
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	n := ergonode.Create(c.NodeName, c.ErlangCookie)
	log.Info().Msg("Started erlang node")

	mongoClient, err := mongo.NewClient(options.Client().ApplyURI(c.MongoURL).SetMaxPoolSize(c.DBPoolSize))
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to mongo %s", err.Error()))
	}

	err = mongoClient.Connect(context.Background())
	if err != nil {
		log.Fatal().Msgf("Error to initialize db connection %s", err)
		return
	}

	repo := mongodb.NewMeTransactionsMongoRepo(mongoClient.Database("medical_data"), c.AuditLogCollectionName)

	srv := service.NewMeTransactionService(c, repo)
	pg2, chn := server.NewGenServer(srv, c)

	n.Spawn(pg2, chn)
	log.Info().Msgf("%+v", pg2.Self)
	log.Info().Msg("Spawned gen server process")
	<-chn
}
