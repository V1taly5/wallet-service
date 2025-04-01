package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	mockrepository "wallet-service/internal/mock/mock_repository"
	"wallet-service/internal/models"
	"wallet-service/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestWalletService_CreateWallet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := mockrepository.NewMockWalletRepository(ctrl)
		mockRepo.EXPECT().
			CreateWallet(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, id uuid.UUID) (*models.Wallet, error) {
				return &models.Wallet{ID: id}, nil
			})

		s := NewWalletService(mockRepo, slog.Default())
		wallet, err := s.CreateWallet(context.Background())

		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, wallet.ID)
	})

	t.Run("repository error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := mockrepository.NewMockWalletRepository(ctrl)
		mockRepo.EXPECT().
			CreateWallet(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("db error"))

		s := NewWalletService(mockRepo, slog.Default())
		wallet, err := s.CreateWallet(context.Background())

		assert.ErrorContains(t, err, "failed to create wallet")
		assert.Nil(t, wallet)
	})
}

func TestWalletService_GetWallet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		id := uuid.New()
		expected := &models.Wallet{ID: id}

		mockRepo := mockrepository.NewMockWalletRepository(ctrl)
		mockRepo.EXPECT().
			GetWallet(gomock.Any(), id).
			Return(expected, nil)

		s := NewWalletService(mockRepo, slog.Default())
		wallet, err := s.GetWallet(context.Background(), id)

		assert.NoError(t, err)
		assert.Equal(t, expected, wallet)
	})

	t.Run("not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		id := uuid.New()
		mockRepo := mockrepository.NewMockWalletRepository(ctrl)
		mockRepo.EXPECT().
			GetWallet(gomock.Any(), id).
			Return(nil, repository.ErrWalletNotFound)

		s := NewWalletService(mockRepo, slog.Default())
		wallet, err := s.GetWallet(context.Background(), id)

		assert.ErrorIs(t, err, ErrInvalidInput)
		assert.Nil(t, wallet)
	})

	t.Run("repository error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		id := uuid.New()
		mockRepo := mockrepository.NewMockWalletRepository(ctrl)
		mockRepo.EXPECT().
			GetWallet(gomock.Any(), id).
			Return(nil, errors.New("db error"))

		s := NewWalletService(mockRepo, slog.Default())
		wallet, err := s.GetWallet(context.Background(), id)

		assert.ErrorContains(t, err, "failed to retrieve wallet")
		assert.Nil(t, wallet)
	})
}

func TestWalletService_ProcessOperation(t *testing.T) {
	validOp := models.WalletOperation{
		WalletID:      uuid.New(),
		OperationType: models.OperationTypeWithdraw,
		Amount:        100,
	}

	t.Run("validation error", func(t *testing.T) {
		s := NewWalletService(nil, slog.Default())
		invalidOp := validOp
		invalidOp.Amount = -100

		_, err := s.ProcessOperation(context.Background(), invalidOp)

		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("success on first try", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := mockrepository.NewMockWalletRepository(ctrl)
		mockRepo.EXPECT().
			UpdateWalletBalance(gomock.Any(), validOp.WalletID, validOp.Amount, validOp.OperationType).
			Return(&models.Wallet{ID: validOp.WalletID}, nil)

		s := NewWalletService(mockRepo, slog.Default())
		_, err := s.ProcessOperation(context.Background(), validOp)

		assert.NoError(t, err)
	})

	t.Run("not found error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := mockrepository.NewMockWalletRepository(ctrl)
		mockRepo.EXPECT().
			UpdateWalletBalance(gomock.Any(), validOp.WalletID, validOp.Amount, validOp.OperationType).
			Return(nil, repository.ErrWalletNotFound)

		s := NewWalletService(mockRepo, slog.Default())
		_, err := s.ProcessOperation(context.Background(), validOp)

		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("max retries exceeded", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := mockrepository.NewMockWalletRepository(ctrl)
		mockRepo.EXPECT().
			UpdateWalletBalance(gomock.Any(), validOp.WalletID, validOp.Amount, validOp.OperationType).
			Times(5).
			Return(nil, errors.New("transient error"))

		s := NewWalletService(mockRepo, slog.Default())
		_, err := s.ProcessOperation(context.Background(), validOp)

		assert.ErrorContains(t, err, "failed to process operation after multiple retries")
	})
}

func TestValidateOperatioеn(t *testing.T) {
	tests := []struct {
		name      string
		operation models.WalletOperation
		wantErr   error
	}{
		{
			name: "valid deposit",
			operation: models.WalletOperation{
				Amount:        100,
				OperationType: models.OperationTypeDeposit,
			},
			wantErr: nil,
		},
		{
			name: "valid withdraw",
			operation: models.WalletOperation{
				Amount:        50,
				OperationType: models.OperationTypeWithdraw,
			},
			wantErr: nil,
		},
		{
			name: "negative amount",
			operation: models.WalletOperation{
				Amount:        -10,
				OperationType: models.OperationTypeDeposit,
			},
			wantErr: ErrAmountMustBePositive,
		},
		{
			name: "zero amount",
			operation: models.WalletOperation{
				Amount:        0,
				OperationType: models.OperationTypeWithdraw,
			},
			wantErr: ErrAmountMustBePositive,
		},
		{
			name: "invalid operation type",
			operation: models.WalletOperation{
				Amount:        100,
				OperationType: "InvalidType",
			},
			wantErr: ErrInvalidOperationType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := validateOperation(tt.operation)
			if !errors.Is(gotErr, tt.wantErr) {
				require.Equal(t, gotErr, tt.wantErr)
				// t.Errorf("validateOperation() = %v, want %v", gotErr, tt.wantErr)
			}
		})
	}
}
