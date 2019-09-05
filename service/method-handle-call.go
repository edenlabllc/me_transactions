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
		logger.Warn().Msgf("Failed to start session: %s", err)
		return "Failed to start session", err
	}
	logger.Debug().Msgf("Started session")

	queryFunc := func(sctx mongo.SessionContext) (string, error) {
		for _, operation := range req.Operations {
			logger.Debug().Msgf("Processing %s in %s collection", operation.Operation, operation.Collection)
			auditLog := &models.ModelAuditLog{
				Collection: operation.Collection,
				ActorID:    req.ActorID,
				InsertedAt: time.Now(),
			}
			switch operation.Operation {
			case "insert":
				rsp, err := srv.processInsert(sctx, auditLog, &operation, req.PatientID, logger)
				if err != nil {
					return rsp, err
				}
			case "update_one":
				rsp, err := srv.processUpdate(sctx, auditLog, &operation, req.PatientID, logger)
				if err != nil {
					return rsp, err
				}
			case "upsert_one":
				rsp, err := srv.processUpsert(sctx, auditLog, &operation, req.PatientID, logger)
				if err != nil {
					return rsp, err
				}
			case "delete_one":
				rsp, err := srv.processDelete(sctx, auditLog, &operation, req.PatientID, logger)
				if err != nil {
					return rsp, err
				}
			default:
				logger.Info().Msgf("Invalid operation")
				continue
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

func (srv *MeTransactionService) processInsert(sctx mongo.SessionContext, auditLog *models.ModelAuditLog, op *entity.Operation, patientID string, logger zerolog.Logger) (string, error) {
	set, err := base64.StdEncoding.DecodeString(op.Set)
	if err != nil {
		return fmt.Sprintf("Invalid base64 string. %v", err.Error()), err
	}

	var s interface{}
	if err := bson.Unmarshal(set, &s); err != nil {
		return fmt.Sprintf("Invalid bson format for Set. %v", err.Error()), err
	}
	ir, err := srv.repo.InsertToProvidedCollection(sctx, op.Collection, s)
	if err != nil {
		return fmt.Sprintf("Aborting transaction. %v", err.Error()), err
	}
	auditLog.PatientID = patientID
	auditLog.Params = s
	auditLog.Type = "INSERT"
	logger.Info().Msgf("Inserted: %+v", ir)
	return srv.processAuditLog(sctx, auditLog)
}

func (srv *MeTransactionService) processUpdate(sctx mongo.SessionContext, auditLog *models.ModelAuditLog, op *entity.Operation, patientID string, logger zerolog.Logger) (string, error) {
	set, err := base64.StdEncoding.DecodeString(op.Set)
	if err != nil {
		return fmt.Sprintf("Invalid base64 string. %v", err.Error()), err
	}
	filter, err := base64.StdEncoding.DecodeString(op.Filter)
	if err != nil {
		return fmt.Sprintf("Invalid base64 string. %v", err.Error()), err
	}

	var s interface{}
	if err := bson.Unmarshal(set, &s); err != nil {
		return fmt.Sprintf("Invalid bson format for Set. %v", err.Error()), err
	}

	var f interface{}
	if err := bson.Unmarshal(filter, &f); err != nil {
		return fmt.Sprintf("Invalid bson format for Filter. %v", err.Error()), err
	}

	ur, err := srv.repo.UpdateProvidedCollection(sctx, op.Collection, f, s)
	if err != nil {
		return fmt.Sprintf("Aborting transaction. %v", err.Error()), err
	}
	auditLog.PatientID = patientID
	auditLog.Params = s
	auditLog.Filter = f
	if ur.ModifiedCount > 0 {
		auditLog.Type = "UPDATE"
	} else {
		auditLog.Type = "INSERT"
	}
	logger.Info().Msgf("Matched: %d, Modified: %d", ur.MatchedCount, ur.ModifiedCount)
	return srv.processAuditLog(sctx, auditLog)
}

func (srv *MeTransactionService) processUpsert(sctx mongo.SessionContext, auditLog *models.ModelAuditLog, op *entity.Operation, patientID string, logger zerolog.Logger) (string, error) {
	set, err := base64.StdEncoding.DecodeString(op.Set)
	if err != nil {
		return fmt.Sprintf("Invalid base64 string. %v", err.Error()), err
	}
	filter, err := base64.StdEncoding.DecodeString(op.Filter)
	if err != nil {
		return fmt.Sprintf("Invalid base64 string. %v", err.Error()), err
	}

	var s interface{}
	if err := bson.Unmarshal(set, &s); err != nil {
		return fmt.Sprintf("Invalid bson format for Set. %v", err.Error()), err
	}

	var f interface{}
	if err := bson.Unmarshal(filter, &f); err != nil {
		return fmt.Sprintf("Invalid bson format for Filter. %v", err.Error()), err
	}
	var upsert = true
	var upsertOptions = &options.UpdateOptions{Upsert: &upsert}
	ur, err := srv.repo.UpdateProvidedCollection(sctx, op.Collection, f, s, upsertOptions)
	if err != nil {
		return fmt.Sprintf("Aborting transaction. %v", err.Error()), err
	}
	if ur.ModifiedCount > 0 {
		auditLog.Type = "UPDATE"
	} else {
		auditLog.Type = "INSERT"
	}
	auditLog.PatientID = patientID
	auditLog.Params = s
	auditLog.Filter = f
	logger.Info().Msgf("Matched: %d, Modified: %d, Upserted: %d", ur.MatchedCount, ur.ModifiedCount, ur.UpsertedCount)
	return srv.processAuditLog(sctx, auditLog)
}

func (srv *MeTransactionService) processDelete(sctx mongo.SessionContext, auditLog *models.ModelAuditLog, op *entity.Operation, patientID string, logger zerolog.Logger) (string, error) {
	filter, err := base64.StdEncoding.DecodeString(op.Filter)
	if err != nil {
		return fmt.Sprintf("Invalid base64 string. %v", err.Error()), err
	}
	var f interface{}
	if err := bson.Unmarshal(filter, &f); err != nil {
		return fmt.Sprintf("Invalid bson format for Filter. %v", err.Error()), err
	}
	dr, err := srv.repo.DeleteFromProvidedCollection(sctx, op.Collection, f)
	if err != nil {
		return fmt.Sprintf("Aborting transaction. %v", err.Error()), err
	}
	auditLog.Filter = f
	auditLog.Type = "DELETE"
	logger.Info().Msgf("Deleted: %d", dr.DeletedCount)
	return srv.processAuditLog(sctx, auditLog)

}

func (srv *MeTransactionService) processAuditLog(sctx mongo.SessionContext, auditLog *models.ModelAuditLog) (string, error) {
	if srv.config.AuditLogEnabled {
		if err := srv.repo.SaveAuditLog(sctx, auditLog); err != nil {
			return fmt.Sprintf("Aborting transaction. %v", err.Error()), err
		}
	}
	return "", nil
}
