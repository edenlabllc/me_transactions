package repo

import (
	"me_transactions/repo/models"

	"go.mongodb.org/mongo-driver/mongo/options"

	"go.mongodb.org/mongo-driver/mongo"
)

type IRepo interface {
	SaveAuditLog(sctx mongo.SessionContext, log *models.ModelAuditLog) error
	InsertToProvidedCollection(sctx mongo.SessionContext, collection string, data interface{}) (*mongo.InsertOneResult, error)
	UpdateProvidedCollection(sctx mongo.SessionContext, collection string, filter interface{}, data interface{}, updateOpts ...*options.UpdateOptions) (*mongo.UpdateResult, error) // Also used as upsert
	DeleteFromProvidedCollection(sctx mongo.SessionContext, collection string, filter interface{}) (*mongo.DeleteResult, error)
	StartSession() (mongo.Session, error)
}
