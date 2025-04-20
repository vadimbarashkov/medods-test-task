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

func (r *RefreshTokenRepository) Save(ctx context.Context, userID, jti uuid.UUID, hashedToken string) error {
	query := `
		INSERT INTO refresh_tokens (user_id, jti, hashed_token)
		VALUES ($1, $2, $3)
	`

	if _, err := exec(ctx, r.pool, query, userID, jti, hashedToken); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationErrCode {
			return repository.ErrRefreshTokenExists
		}
		return fmt.Errorf("execute query: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepository) Get(ctx context.Context, userID, jti uuid.UUID) (string, bool, error) {
	query := `
		SELECT hashed_token, revoked
		FROM refresh_tokens
		WHERE user_id = $1 AND jti = $2
	`

	var (
		hashedToken string
		revoked     bool
	)

	row := queryRow(ctx, r.pool, query, userID, jti)
	if err := row.Scan(&hashedToken, &revoked); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, repository.ErrRefreshTokenNotFound
		}
		return "", false, fmt.Errorf("execute query: %w", err)
	}

	return hashedToken, revoked, nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, userID, jti uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked = TRUE
		WHERE user_id = $1 AND jti = $2
	`

	result, err := exec(ctx, r.pool, query, userID, jti)
	if err != nil {
		return fmt.Errorf("execute query: %w", err)
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
			return fmt.Errorf("rollback transaction: %w", errTx)
		}
		return fmt.Errorf("execute operation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
