package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"wallet-service/internal/models"

	"github.com/google/uuid"
)

var (
	ErrWalletNotFound         = errors.New("wallet not found")
	ErrInsufficientFunds      = errors.New("insufficient funds")
	ErrConcurrentModification = errors.New("concurrent modification detected")
	ErrUnknownOperationType   = errors.New("unknown operation type")
)

type WalletRepository struct {
	db *sql.DB
}

func NewWalletRepository(db *sql.DB) *WalletRepository {
	return &WalletRepository{
		db: db,
	}
}

func (r *WalletRepository) CreateWallet(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	wallet := &models.Wallet{
		ID:        id,
		Balance:   0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
	}

	query := `INSERT INTO wallets (id, balance, created_at, updated_at, version) 
				 VALUES ($1, $2, $3, $4, $5) 
				 RETURNING id, balance, created_at, updated_at, version`

	err := r.db.QueryRowContext(
		ctx,
		query,
		wallet.ID,
		wallet.Balance,
		wallet.CreatedAt,
		wallet.UpdatedAt,
		wallet.Version,
	).Scan(
		&wallet.ID,
		&wallet.Balance,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
		&wallet.Version,
	)

	if err != nil {
		return nil, err
	}

	return wallet, nil
}

func (r *WalletRepository) GetWallet(ctx context.Context, id uuid.UUID) (*models.Wallet, error) {
	query := `SELECT id, balance, created_at, updated_at, version FROM wallets WHERE id = $1`
	wallet := &models.Wallet{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&wallet.ID,
		&wallet.Balance,
		&wallet.CreatedAt,
		&wallet.UpdatedAt,
		&wallet.Version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, err
	}

	return wallet, nil
}

func (r *WalletRepository) UpdateWalletBalance(ctx context.Context, id uuid.UUID, amount int64,
	operation models.OperationType) (*models.Wallet, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	query := `SELECT id, balance, created_at, updated_at, version FROM wallets WHERE id = $1 FOR UPDATE`

	wallet := models.Wallet{}
	err = tx.QueryRowContext(ctx, query, id).Scan(
		&wallet.ID, &wallet.Balance, &wallet.CreatedAt, &wallet.UpdatedAt, &wallet.Version,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, err
	}

	newBalance := wallet.Balance
	switch operation {
	case models.OperationTypeWithdraw:
		if wallet.Balance < amount {
			return nil, ErrInsufficientFunds
		}
		newBalance -= amount
	case models.OperationTypeDeposit:
		newBalance += amount
	default:
		return nil, ErrUnknownOperationType
	}

	updateQuery := `UPDATE wallets SET balance = $1, updated_at = $2, version = version + 1
	WHERE id = $3 AND version = $4
	RETURNING id, balance, created_at, updated_at, version`

	updatedWallet := &models.Wallet{}
	err = tx.QueryRowContext(
		ctx,
		updateQuery,
		newBalance,
		time.Now(),
		id,
		wallet.Version,
	).Scan(
		&updatedWallet.ID,
		&updatedWallet.Balance,
		&updatedWallet.CreatedAt,
		&updatedWallet.UpdatedAt,
		&updatedWallet.Version,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrConcurrentModification
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return updatedWallet, nil
}

func (r *WalletRepository) CreateTabeIfNotExists(ctx context.Context) error {
	query := `CREATE TABLE IF NOT EXISTS wallets (
					id UUID PRIMARY KEY,
					balance BIGINT NOT NULL DEFAULT 0,
					created_at TIMESTAMP NOT NULL,
					updated_at TIMESTAMP NOT NULL,
					version INTEGER NOT NULL DEFAULT 1
				)`
	_, err := r.db.ExecContext(ctx, query)
	return err
}
