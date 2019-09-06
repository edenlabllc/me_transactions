package service

import (
	"context"
	"me_transactions/cfg"
	"me_transactions/repo"
	"me_transactions/server/entity"

	"github.com/rs/zerolog"
)

type IMeTransactionService interface {
	HandleHealthCheck(ctx context.Context, req string) error
	HandleCall(ctx context.Context, req *entity.Request, logger zerolog.Logger) (string, error) // TODO: rename and split
}

type MeTransactionService struct {
	repo   repo.IRepo
	config *cfg.Config
}

func NewMeTransactionService(config *cfg.Config, repo repo.IRepo) IMeTransactionService {
	return &MeTransactionService{
		repo:   repo,
		config: config,
	}
}
