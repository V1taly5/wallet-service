package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"wallet-service/internal/models"
	"wallet-service/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrAmountMustBePositive = errors.New("amount must be positive")
	ErrInvalidOperationType = errors.New("invalid operation type")
	ErrInvalidInput         = errors.New("invalid input")
	ErrOperationFailed      = errors.New("operation failed")
)

type WalletService struct {
	repo WalletRepository
	log  *slog.Logger
}

func NewWalletService(repo WalletRepository, log *slog.Logger) *WalletService {
	return &WalletService{
		repo: repo,
		log:  log,
	}
}

func (s *WalletService) CreateWallet(ctx context.Context) (*models.Wallet, error) {
	op := "service.CreateWallet"
	log := s.log.With(slog.String("op", op))

	id, err := uuid.NewRandom()
	if err != nil {
		log.Error("failed to generate wallet ID", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
		return nil, fmt.Errorf("failed to generate wallet ID: %w", err)
	}
	wallet, err := s.repo.CreateWallet(ctx, id)
	if err != nil {
		log.Error("failed to create wallet", slog.String("wallet_id", id.String()), slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}
	log.Info("wallet created successfully", slog.String("wallet_id", wallet.ID.String()))
	return wallet, nil
}

func (s *WalletService) GetWallet(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	op := "service.GetWallet"
	log := s.log.With(slog.String("op", op), slog.String("wallet_id", id.String()))

	wallet, err := s.repo.GetWallet(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrWalletNotFound) {
			log.Warn("wallet not found")
			return nil, ErrInvalidInput
		}
		log.Error("failed to retrieve wallet", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
		return nil, fmt.Errorf("failed to retrieve wallet: %w", err)
	}
	log.Info("wallet retrieved successfully")
	return wallet, nil
}

func (s *WalletService) ProcessOperation(ctx context.Context, operation models.WalletOperation) (*models.Wallet, error) {
	op := "service.ProcessOperation"
	log := s.log.With(slog.String("op", op), slog.String("wallet_id", operation.WalletID.String()), slog.String("operation", string(operation.OperationType)))

	if err := validateOperation(operation); err != nil {
		log.Warn("invalid operation", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
		return nil, ErrInvalidInput
	}

	maxRetries := 5
	var lastErr error
	backoff := 10 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		wallet, err := s.repo.UpdateWalletBalance(ctx, operation.WalletID, operation.Amount, operation.OperationType)
		if err == nil {
			log.Info("operation processed successfully")
			return wallet, nil
		}

		if errors.Is(err, repository.ErrWalletNotFound) || errors.Is(err, repository.ErrInsufficientFunds) {
			log.Warn("operation failed due to invalid input", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
			return nil, ErrInvalidInput
		}

		lastErr = err
		// exponential delay
		time.Sleep(backoff)
		backoff *= 2
	}

	return nil, fmt.Errorf("failed to process operation after multiple retries: %w", lastErr)
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
