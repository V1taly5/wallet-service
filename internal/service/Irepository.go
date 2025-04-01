package service

import (
	"context"
	"wallet-service/internal/models"

	"github.com/google/uuid"
)

type WalletRepository interface {
	CreateWallet(context.Context, uuid.UUID) (*models.Wallet, error)
	GetWallet(context.Context, uuid.UUID) (*models.Wallet, error)
	UpdateWalletBalance(context.Context, uuid.UUID, int64, models.OperationType) (*models.Wallet, error)
}
