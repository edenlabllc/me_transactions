package main

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoURL   string
	dbPoolSize uint64
	//dbWriteConcern int
)

func GetMongodbDSN() string {
	//return fmt.Sprintf("mongodb://%s:%s/?replicaSet=%s",
	//	"mongo4", "27017", "rs0")
	return "mongodb://localhost:27017/concurrency?replicaSet=rs0"
}

var DBCollTx = "transactions-info"
var DBCollBI = "block-info"

type TransactionInfo struct {
	ID primitive.ObjectID `bson:"_id,omitempty"`
	//BlockNumber *primitive.ObjectID `bson:"block_number"`
	Hash string `bson:"hash"`
	//Amount      string              `bson:"amount"`
}

type BlockInfo struct {
	ID     primitive.ObjectID `bson:"_id"`
	Number int                `bson:"number"`
}

type txRepo struct {
	db *mongo.Database
}

func NewRepo(c *mongo.Client) *txRepo {
	db := c.Database("concurrency")
	return &txRepo{db: db}
}

func (r *txRepo) InsertTransactionInfo(sctx mongo.SessionContext, i *TransactionInfo) error {
	log.Info().Msg("Inserting tx info")
	if _, err := r.db.Collection(DBCollTx).
		InsertOne(sctx, i); err != nil {
		return err
	}

	return nil
}

func (r *txRepo) InsertBlockInfo(sctx mongo.SessionContext, i *BlockInfo) error {
	log.Info().Msg("Inserting block info")
	if _, err := r.db.Collection(DBCollBI).
		InsertOne(sctx, i); err != nil {
		return err
	}

	return nil
}

//
//
//func main() {
//	ctx := context.Background()
//	client, err := mongo.NewClient(options.Client().ApplyURI(GetMongodbDSN()).SetMaxPoolSize(10))
//	if err != nil {
//		log.Error().Err(err).Msgf("Error on [NewClient]. Mongo labels: %v", returnMongoErrLabel(err))
//		return
//	}
//	err = client.Connect(ctx)
//	if err != nil {
//		log.Error().Err(err).Msgf("Error on [Connect]. Mongo labels: %v", returnMongoErrLabel(err))
//		return
//	}
//
//	wg := new(sync.WaitGroup)
//	for i := 0; i < 1000; i++ {
//		wg.Add(1)
//		go func() {
//			defer wg.Done()
//			session, _ := client.StartSession()
//			repo := NewRepo(client)
//
//			// set concern
//
//			err = session.StartTransaction()
//			if err != nil {
//				log.Error().Err(err).Msgf("Error on [StartTransaction]. Mongo labels: %v", returnMongoErrLabel(err))
//			}
//
//			err = mongo.WithSession(ctx, session, func(sctx mongo.SessionContext) error {
//				return runTxWithRetry(sctx, repo)
//			})
//
//			if err != nil {
//				log.Error().Err(err).Msgf("Error on [runQuery]. Mongo labels: %v", returnMongoErrLabel(err))
//
//			}
//			session.EndSession(ctx)
//			return
//		}()
//	}
//	wg.Wait()
//	return
//}
//
//func runTxWithRetry(sctx mongo.SessionContext, repo *txRepo) error {
//	for {
//		err := runQuery(sctx, repo)
//		switch e := err.(type) {
//		case nil:
//			return nil
//		case mongo.CommandError:
//			if e.HasErrorLabel("UnknownTransactionCommitResult") {
//				continue
//			}
//			if e.HasErrorLabel("TransientTransactionError") {
//				continue
//			}
//			log.Info().Msgf("Retry transaction: %s", err)
//			return err
//		default:
//			return err
//		}
//	}
//}

func runQuery(sctx mongo.SessionContext, repo *txRepo) (interface{}, error) {
	rand.Seed(time.Now().UnixNano())

	err := repo.InsertTransactionInfo(sctx, &TransactionInfo{
		ID:   primitive.NewObjectID(),
		Hash: "123",
	})
	if err != nil {
		sctx.AbortTransaction(sctx)
		return nil, err
	}
	d := time.Duration(rand.Int63n(5)) * time.Second
	time.Sleep(d)

	err = repo.InsertBlockInfo(sctx, &BlockInfo{
		ID:     primitive.NewObjectID(),
		Number: rand.Intn(100),
	})
	if err != nil {
		sctx.AbortTransaction(sctx)
		return nil, err
	}
	err = sctx.CommitTransaction(sctx)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func returnMongoErrLabel(err error) string {
	errs := ""
	switch e := err.(type) {
	case mongo.CommandError:
		for _, l := range e.Labels {
			errs += l + ", "
		}
	}
	return errs
}

func main() {
	ctx := context.Background()
	client, err := mongo.NewClient(options.Client().ApplyURI(GetMongodbDSN()).SetMaxPoolSize(10))
	if err != nil {
		log.Error().Err(err).Msgf("Error on [NewClient]. Mongo labels: %v", returnMongoErrLabel(err))
		return
	}
	err = client.Connect(ctx)
	if err != nil {
		log.Error().Err(err).Msgf("Error on [Connect]. Mongo labels: %v", returnMongoErrLabel(err))
		return
	}
	session, _ := client.StartSession()
	wg := new(sync.WaitGroup)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			repo := NewRepo(client)

			_, err := session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {
				return runQuery(sctx, repo)
			})

			if err != nil {
				log.Error().Err(err).Msgf("Error on [StartTransaction]. Mongo labels: %v", returnMongoErrLabel(err))
			}

			session.EndSession(ctx)
			return
		}()
	}
	wg.Wait()
	return
}
