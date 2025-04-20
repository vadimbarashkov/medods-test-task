package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrRefreshTokenExists   = errors.New("refresh token exists")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)

type RefreshTokenRepository interface {
	Save(ctx context.Context, userID uuid.UUID, hashedToken string) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (string, bool, error)
	Revoke(ctx context.Context, userID uuid.UUID) error
	Transaction(ctx context.Context, fn func(context.Context) error) error
}
