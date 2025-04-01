package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"
	"wallet-service/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var log = slog.New(slog.NewTextHandler(os.Stdin, &slog.HandlerOptions{Level: slog.LevelInfo}))

func TestWalletRepository_CreateWallet_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewWalletRepository(db, log)
	ctx := context.Background()
	testID := uuid.New()
	now := time.Now().UTC()

	mock.ExpectQuery(`^INSERT INTO wallets`).
		WithArgs(
			testID,
			0,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			1,
		).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "balance", "created_at", "updated_at", "version"}).
				AddRow(testID, 0, now, now, 1),
		)

	wallet, err := repo.CreateWallet(ctx, testID)

	require.NoError(t, err)
	assert.NotNil(t, wallet)

	assert.Equal(t, testID, wallet.ID)
	assert.Equal(t, int64(0), wallet.Balance)
	assert.Equal(t, 1, wallet.Version)

	assert.WithinDuration(t, now, wallet.CreatedAt, 2*time.Second)
	assert.WithinDuration(t, now, wallet.UpdatedAt, 2*time.Second)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWalletRepository_CreateWallet_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewWalletRepository(db, log)
	testID := uuid.New()

	// Мокирование ошибки
	mock.ExpectQuery(`^INSERT INTO wallets`).
		WillReturnError(sql.ErrConnDone)

	wallet, err := repo.CreateWallet(context.Background(), testID)

	// Проверки
	require.Error(t, err)
	assert.Nil(t, wallet)
	assert.ErrorIs(t, err, sql.ErrConnDone)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWalletRepository_CreateWallet_ContextCanceled(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewWalletRepository(db, log)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Ожидаем, что запрос даже не будет выполнен
	mock.ExpectQuery(`^INSERT INTO wallets`).
		WillReturnError(context.Canceled)

	wallet, err := repo.CreateWallet(ctx, uuid.New())

	require.Error(t, err)
	assert.Nil(t, wallet)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestWalletRepository_GetWallet_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewWalletRepository(db, log)
	testID := uuid.New()
	now := time.Now().UTC()

	// Мокируем успешный SELECT
	mock.ExpectQuery(`^SELECT id, balance, created_at, updated_at, version FROM wallets WHERE id = \$1$`).
		WithArgs(testID).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "balance", "created_at", "updated_at", "version"}).
				AddRow(testID, 100, now, now, 2),
		)

	wallet, err := repo.GetWallet(context.Background(), testID)

	require.NoError(t, err)
	assert.Equal(t, testID, wallet.ID)
	assert.Equal(t, int64(100), wallet.Balance)
	assert.Equal(t, 2, wallet.Version)
	assert.WithinDuration(t, now, wallet.CreatedAt, time.Second)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWalletRepository_GetWallet_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewWalletRepository(db, log)
	testID := uuid.New()

	mock.ExpectQuery(`^SELECT`).
		WithArgs(testID).
		WillReturnError(sql.ErrNoRows)

	wallet, err := repo.GetWallet(context.Background(), testID)

	require.ErrorIs(t, err, ErrWalletNotFound)
	assert.Nil(t, wallet)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWalletRepository_GetWallet_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewWalletRepository(db, log)
	testID := uuid.New()

	expectedErr := errors.New("connection failed")
	mock.ExpectQuery(`^SELECT`).
		WithArgs(testID).
		WillReturnError(expectedErr)

	wallet, err := repo.GetWallet(context.Background(), testID)

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, wallet)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWalletRepository_GetWallet_ContextCanceled(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewWalletRepository(db, log)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	testID := uuid.New()

	mock.ExpectQuery(`^SELECT`).
		WithArgs(testID).
		WillReturnError(context.Canceled)

	wallet, err := repo.GetWallet(ctx, testID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
	assert.Nil(t, wallet)
}

func TestUpdateWalletBalance_DepositSuccess(t *testing.T) {
	db, mock, _ := sqlmock.New()
	repo := NewWalletRepository(db, log)
	defer db.Close()

	testID := uuid.New()
	initialBalance := 100
	depositAmount := 50

	mock.ExpectBegin()

	mock.ExpectQuery(`SELECT .* FOR UPDATE`).
		WithArgs(testID).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "balance", "created_at", "updated_at", "version"}).
				AddRow(testID, initialBalance, time.Now(), time.Now(), 1),
		)

	mock.ExpectQuery(`UPDATE wallets`).
		WithArgs(initialBalance+depositAmount, sqlmock.AnyArg(), testID, 1).
		WillReturnRows(
			sqlmock.NewRows([]string{"id", "balance", "created_at", "updated_at", "version"}).
				AddRow(testID, initialBalance+depositAmount, time.Now(), time.Now(), 2),
		)

	mock.ExpectCommit()

	result, err := repo.UpdateWalletBalance(
		context.Background(),
		testID,
		int64(depositAmount),
		models.OperationTypeDeposit,
	)

	require.NoError(t, err)
	assert.Equal(t, int64(initialBalance+depositAmount), result.Balance)
	assert.Equal(t, 2, result.Version)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateWalletBalance_WithdrawSuccess(t *testing.T) {
	db, mock, _ := sqlmock.New()
	repo := NewWalletRepository(db, log)
	defer db.Close()

	testID := uuid.New()
	initialBalance := 100
	withdrawAmount := 30

	mock.ExpectBegin()

	mock.ExpectQuery(`SELECT .* FOR UPDATE`).
		WithArgs(testID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "balance", "created_at", "updated_at", "version"}).
			AddRow(testID, initialBalance, time.Now(), time.Now(), 1),
		)

	mock.ExpectQuery(`UPDATE wallets`).
		WithArgs(initialBalance-withdrawAmount, sqlmock.AnyArg(), testID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "balance", "created_at", "updated_at", "version"}).
			AddRow(testID, initialBalance-withdrawAmount, time.Now(), time.Now(), 2),
		)

	mock.ExpectCommit()

	result, err := repo.UpdateWalletBalance(
		context.Background(),
		testID,
		int64(withdrawAmount),
		models.OperationTypeWithdraw,
	)

	require.NoError(t, err)
	assert.Equal(t, int64(initialBalance-withdrawAmount), result.Balance)
	assert.Equal(t, 2, result.Version)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateWalletBalance_InsufficientFunds(t *testing.T) {
	db, mock, _ := sqlmock.New()
	repo := NewWalletRepository(db, log)
	defer db.Close()

	testID := uuid.New()
	initialBalance := 50
	withdrawAmount := 100

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT .* FOR UPDATE`).
		WithArgs(testID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "balance", "created_at", "updated_at", "version"}).
			AddRow(testID, initialBalance, time.Now(), time.Now(), 1))

	_, err := repo.UpdateWalletBalance(
		context.Background(),
		testID,
		int64(withdrawAmount),
		models.OperationTypeWithdraw,
	)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrInsufficientFunds)
	assert.NoError(t, mock.ExpectationsWereMet())
}
