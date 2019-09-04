package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"me_transactions/repo/models"
	"me_transactions/server/entity"
)

func (srv *MeTransactionService) HandleCall(ctx context.Context, req *entity.Request, logger zerolog.Logger) (string, error) {
	concern := writeconcern.WMajority()
	if srv.config.DBWriteConcern != 0 {
		concern = writeconcern.W(srv.config.DBWriteConcern)
	}
	txOpts := options.Transaction().SetWriteConcern(writeconcern.New(concern))
	session, err := srv.repo.StartSession()
	if err != nil {
		return "", err
	}

	queryFunc := func(sctx mongo.SessionContext) (string, error) {
		for _, operation := range req.Operations {
			auditLog := &models.ModelAuditLog{
				Collection: operation.Collection,
				ActorID:    req.ActorID,
				InsertedAt: time.Now(),
			}

			set, err := base64.StdEncoding.DecodeString(operation.Set)
			if err != nil {
				return fmt.Sprintf("Invalid base64 string. %v", err.Error()), err
			}
			filter, err := base64.StdEncoding.DecodeString(operation.Filter)
			if err != nil {
				return fmt.Sprintf("Invalid base64 string. %v", err.Error()), err
			}

			var s interface{}
			if len(set) > 0 {
				if err := bson.Unmarshal(set, &s); err != nil {
					return fmt.Sprintf("Invalid bson format for Set. %v", err.Error()), err
				}
			}

			var f interface{}
			if len(filter) > 0 {
				if err := bson.Unmarshal(filter, &f); err != nil {
					return fmt.Sprintf("Invalid bson format for Filter. %v", err.Error()), err
				}
			}

			switch operation.Operation {
			case "insert":
				if err := srv.repo.InsertToProvidedCollection(sctx, operation.Collection, s); err != nil {
					return fmt.Sprintf("Aborting transaction. %v", err.Error()), err
				}
				auditLog.PatientID = req.PatientID
				auditLog.Params = s
				auditLog.Type = "INSERT"
			case "update_one":
				ur, err := srv.repo.UpdateProvidedCollection(sctx, operation.Collection, f, s)
				if err != nil {
					return fmt.Sprintf("Aborting transaction. %v", err.Error()), err
				}
				auditLog.PatientID = req.PatientID
				auditLog.Params = s
				auditLog.Filter = f
				if ur.ModifiedCount > 0 {
					auditLog.Type = "UPDATE"
				} else {
					auditLog.Type = "INSERT"
				}
			case "upsert_one":
				var upsert = true
				var upsertOptions = &options.UpdateOptions{Upsert: &upsert}
				ur, err := srv.repo.UpdateProvidedCollection(sctx, operation.Collection, f, s, upsertOptions)
				if err != nil {
					return fmt.Sprintf("Aborting transaction. %v", err.Error()), err
				}
				if ur.ModifiedCount > 0 {
					auditLog.Type = "UPDATE"
				} else {
					auditLog.Type = "INSERT"
				}
				auditLog.PatientID = req.PatientID
				auditLog.Params = s
				auditLog.Filter = f
			case "delete_one":
				if err := srv.repo.DeleteFromProvidedCollection(sctx, operation.Collection, f); err != nil {

				}
				auditLog.Filter = f
				auditLog.Type = "DELETE"
			default:
				continue
			}
			if srv.config.AuditLogEnabled {
				if err := srv.repo.SaveAuditLog(sctx, auditLog); err != nil {
					return fmt.Sprintf("Aborting transaction. %v", err.Error()), err
				}
			}
		}
		return "", nil
	}

	replString, err := session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {
		return queryFunc(sctx)
	}, txOpts)
	session.EndSession(ctx)
	if err != nil {
		return replString.(string), err
	}

	return "ok", nil
}
