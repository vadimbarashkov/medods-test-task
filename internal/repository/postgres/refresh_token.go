package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vadimbarashkov/medods-test-task/internal/repository"
)

type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

func (r *RefreshTokenRepository) Save(ctx context.Context, userID uuid.UUID, hashedToken string) error {
	query := `
		INSERT INTO refresh_tokens (user_id, hashed_token)
		VALUES ($1, $2)
	`

	var err error
	if tx, ok := txFromContext(ctx); ok {
		_, err = tx.Exec(ctx, query, userID, hashedToken)
	} else {
		_, err = r.pool.Exec(ctx, query, userID, hashedToken)
	}

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationErrCode {
			return repository.ErrRefreshTokenExists
		}
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (string, bool, error) {
	query := `
		SELECT hashed_token, revoked
		FROM refresh_tokens
		WHERE user_id = $1
	`

	var (
		hashedToken string
		revoked     bool
	)

	var err error
	if tx, ok := txFromContext(ctx); ok {
		err = tx.QueryRow(ctx, query, userID).Scan(&hashedToken, &revoked)
	} else {
		err = r.pool.QueryRow(ctx, query, userID).Scan(&hashedToken, &revoked)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, repository.ErrRefreshTokenNotFound
		}
		return "", false, fmt.Errorf("failed to execute query: %w", err)
	}

	return hashedToken, revoked, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked = TRUE
		WHERE user_id = $1
	`

	var (
		result pgconn.CommandTag
		err    error
	)
	if tx, ok := txFromContext(ctx); ok {
		result, err = tx.Exec(ctx, query, userID)
	} else {
		result, err = r.pool.Exec(ctx, query, userID)
	}

	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrRefreshTokenNotFound
	}

	return nil
}

func (r *RefreshTokenRepository) Transaction(ctx context.Context, fn func(context.Context) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(contextWithTx(ctx, tx)); err != nil {
		if errTx := tx.Rollback(ctx); errTx != nil {
			return fmt.Errorf("failed to rollback transaction: %w", errTx)
		}
		return fmt.Errorf("failed to execute operation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
