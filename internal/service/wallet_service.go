package service

import (
	"context"
	"errors"
	"time"
	"wallet-service/internal/models"
	"wallet-service/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrAmountMustBePositive = errors.New("amount must be positive")
	ErrInvalidOperationType = errors.New("invalid operation type")
)

type WalletService struct {
	repo *repository.WalletRepository
}

func NewWalletService(repo *repository.WalletRepository) *WalletService {
	return &WalletService{
		repo: repo,
	}
}

func (s *WalletService) CreateWallet(ctx context.Context) (*models.Wallet, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	return s.repo.CreateWallet(ctx, id)
}

func (s *WalletService) GetWallet(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	return s.repo.GetWallet(ctx, id)
}

func (s *WalletService) ProcessOperation(ctx context.Context, operation models.WalletOperation) (*models.Wallet, error) {
	if err := validateOperation(operation); err != nil {
		return nil, err
	}

	maxRetries := 5
	var lastErr error
	backoff := 10 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		wallet, err := s.repo.UpdateWalletBalance(ctx, operation.WalletID, operation.Amount, operation.OperationType)
		if err == nil {
			return wallet, nil
		}

		if errors.Is(err, repository.ErrWalletNotFound) || errors.Is(err, repository.ErrInsufficientFunds) {
			return nil, err
		}

		lastErr = err
		// exponential delay
		time.Sleep(backoff)
		backoff *= 2
	}

	return nil, errors.New("failed to process operation after multiple retries: " + lastErr.Error())
}

func validateOperation(operation models.WalletOperation) error {
	if operation.Amount <= 0 {
		return ErrAmountMustBePositive
	}
	if operation.OperationType != models.OperationTypeDeposit && operation.OperationType != models.OperationTypeWithdraw {
		return ErrInvalidOperationType
	}
	return nil
}
