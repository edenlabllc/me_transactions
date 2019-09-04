package mongodb

import (
	"me_transactions/repo"
	"me_transactions/repo/models"

	"go.mongodb.org/mongo-driver/mongo/options"

	"go.mongodb.org/mongo-driver/mongo"
)

type meTransactionRepo struct {
	db                 *mongo.Database
	auditLogCollection *mongo.Collection
}

func NewMeTransactionsMongoRepo(db *mongo.Database, auditLogCollection string) repo.IRepo {
	return &meTransactionRepo{db: db, auditLogCollection: db.Collection(auditLogCollection)}
}

func (repo *meTransactionRepo) SaveAuditLog(sctx mongo.SessionContext, log *models.ModelAuditLog) error {
	_, err := repo.auditLogCollection.InsertOne(sctx, log)
	if err != nil {
		return err
	}
	return nil
}

func (repo *meTransactionRepo) InsertToProvidedCollection(sctx mongo.SessionContext, collection string, data interface{}) error {
	_, err := repo.db.Collection(collection).InsertOne(sctx, data)
	if err != nil {
		return err
	}
	return nil

}

func (repo *meTransactionRepo) UpdateProvidedCollection(sctx mongo.SessionContext, collection string, filter interface{}, data interface{}, updateOpts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	ur, err := repo.db.Collection(collection).UpdateOne(sctx, filter, data, updateOpts...)
	if err != nil {
		return nil, err
	}
	return ur, nil

}

func (repo *meTransactionRepo) DeleteFromProvidedCollection(sctx mongo.SessionContext, collection string, filter interface{}) error {
	_, err := repo.db.Collection(collection).DeleteOne(sctx, filter)
	if err != nil {
		return err
	}
	return nil
}

func (repo *meTransactionRepo) StartSession() (mongo.Session, error) {
	return repo.db.Client().StartSession()
}
