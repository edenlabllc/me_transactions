package main

import (
	"context"

	"github.com/halturin/ergonode"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"me_transactions/cfg"
	"me_transactions/repo/mongodb"
	"me_transactions/server"
	"me_transactions/service"
)

func main() {
	n := ergonode.Create("examplenode@127.0.0.1", "123")
	log.Info().Msg("Started erlang node")

	c, _ := cfg.ConfigFromEnv()
	mongoClient, err := mongo.NewClient(options.Client().ApplyURI(c.GetMongodbDSN()).SetMaxPoolSize(10))

	err = mongoClient.Connect(context.Background())
	if err != nil {
		log.Fatal().Msgf("Error to initialize db connection %s", err)
		return
	}

	repo := mongodb.NewMeTransactionsMongoRepo(mongoClient.Database(c.MongoDBName), "audit")

	srv := service.NewMeTransactionService(c, repo)
	pg2, chn := server.NewGenServer(srv)

	n.Spawn(pg2, chn)
	log.Info().Msgf("%+v", pg2.Self)
	log.Info().Msg("Spawned gen server process")
	<-chn
}
