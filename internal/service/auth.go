package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/vadimbarashkov/medods-test-task/internal/entity"
	"github.com/vadimbarashkov/medods-test-task/internal/repository"
	"github.com/vadimbarashkov/medods-test-task/pkg/notifier"
	"golang.org/x/crypto/bcrypt"
)

const tokenIssuer = "auth-service"

var ErrInvalidToken = errors.New("invalid token")

type AuthService struct {
	accessTokenSecret  []byte
	accessTokenTTL     time.Duration
	refreshTokenSecret []byte
	refreshTokenTTL    time.Duration
	refreshTokenRepo   repository.RefreshTokenRepository
	emailNotifier      notifier.EmailNotifier
}

type AuthServiceParams struct {
	AccessTokenSecret  []byte
	AccessTokenTTL     time.Duration
	RefreshTokenSecret []byte
	RefreshTokenTTL    time.Duration
	RefreshTokenRepo   repository.RefreshTokenRepository
	EmailNotifier      notifier.EmailNotifier
}

func NewAuthService(p AuthServiceParams) *AuthService {
	return &AuthService{
		accessTokenSecret:  p.AccessTokenSecret,
		accessTokenTTL:     p.AccessTokenTTL,
		refreshTokenSecret: p.RefreshTokenSecret,
		refreshTokenTTL:    p.RefreshTokenTTL,
		refreshTokenRepo:   p.RefreshTokenRepo,
		emailNotifier:      p.EmailNotifier,
	}
}

type generateTokenParams struct {
	userID   uuid.UUID
	clientIP net.IP
	ttl      time.Duration
	jti      uuid.UUID
	secret   []byte
}

func (s *AuthService) generateToken(p generateTokenParams) (string, error) {
	claims := &entity.TokenClaims{
		UserID:   p.userID.String(),
		ClientIP: p.clientIP.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tokenIssuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(p.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        p.jti.String(),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString(p.secret)
}

func (s *AuthService) parseToken(token string, secret []byte) (*entity.TokenClaims, error) {
	parsedToken, err := jwt.ParseWithClaims(token, &entity.TokenClaims{}, func(t *jwt.Token) (any, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := parsedToken.Claims.(*entity.TokenClaims)
	if !ok || !parsedToken.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (s *AuthService) hashToken(token string) (string, error) {
	sha := sha256.Sum256([]byte(token))
	bytes, err := bcrypt.GenerateFromPassword(sha[:], bcrypt.DefaultCost)
	return string(bytes), err
}

func (s *AuthService) checkTokenHash(token, hashedToken string) bool {
	sha := sha256.Sum256([]byte(token))
	return bcrypt.CompareHashAndPassword([]byte(hashedToken), sha[:]) == nil
}

func (s *AuthService) IssueTokens(ctx context.Context, userID uuid.UUID, clientIP net.IP) (string, string, error) {
	accessToken, err := s.generateToken(generateTokenParams{
		userID:   userID,
		clientIP: clientIP,
		ttl:      s.accessTokenTTL,
		jti:      uuid.New(),
		secret:   s.accessTokenSecret,
	})
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshID := uuid.New()
	refreshToken, err := s.generateToken(generateTokenParams{
		userID:   userID,
		clientIP: clientIP,
		ttl:      s.refreshTokenTTL,
		jti:      refreshID,
		secret:   s.refreshTokenSecret,
	})
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	hashed, err := s.hashToken(refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("hash refresh token: %w", err)
	}

	if err := s.refreshTokenRepo.Save(ctx, userID, refreshID, hashed); err != nil {
		return "", "", fmt.Errorf("save refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string, clientIP net.IP) (string, string, error) {
	claims, err := s.parseToken(refreshToken, s.refreshTokenSecret)
	if err != nil {
		return "", "", fmt.Errorf("parse refresh token: %w", err)
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return "", "", ErrInvalidToken
	}
	refreshID, err := uuid.Parse(claims.ID)
	if err != nil {
		return "", "", ErrInvalidToken
	}

	storedHash, revoked, err := s.refreshTokenRepo.Get(ctx, userID, refreshID)
	if err != nil {
		if errors.Is(err, repository.ErrRefreshTokenNotFound) {
			return "", "", ErrInvalidToken
		}
		return "", "", fmt.Errorf("get refresh token: %w", err)
	}

	if revoked || !s.checkTokenHash(refreshToken, storedHash) || claims.ExpiresAt.Before(time.Now()) {
		return "", "", ErrInvalidToken
	}

	if clientIP.String() != claims.ClientIP {
		err := s.emailNotifier.Send(
			"example-user@gmail.com",
			"Refresh Tokens Warning",
			"We encounterd an issue while refreshing your tokens...",
		)
		if err != nil {
			return "", "", fmt.Errorf("send warning email: %w", err)
		}
	}

	newAccessToken, err := s.generateToken(generateTokenParams{
		userID:   userID,
		clientIP: clientIP,
		ttl:      s.accessTokenTTL,
		jti:      uuid.New(),
		secret:   s.accessTokenSecret,
	})
	if err != nil {
		return "", "", fmt.Errorf("generate new access token: %w", err)
	}

	newRefreshID := uuid.New()
	newRefreshToken, err := s.generateToken(generateTokenParams{
		userID:   userID,
		clientIP: clientIP,
		ttl:      s.refreshTokenTTL,
		jti:      newRefreshID,
		secret:   s.refreshTokenSecret,
	})
	if err != nil {
		return "", "", fmt.Errorf("generate new refresh token: %w", err)
	}

	newHashedRefreshToken, err := s.hashToken(newRefreshToken)
	if err != nil {
		return "", "", fmt.Errorf("hash new refresh token: %w", err)
	}

	err = s.refreshTokenRepo.Transaction(ctx, func(ctx context.Context) error {
		if err := s.refreshTokenRepo.Revoke(ctx, userID, refreshID); err != nil {
			return fmt.Errorf("revoke old refresh token: %w", err)
		}
		if err := s.refreshTokenRepo.Save(ctx, userID, newRefreshID, newHashedRefreshToken); err != nil {
			return fmt.Errorf("save new refresh token: %w", err)
		}
		return nil
	})
	if err != nil {
		return "", "", fmt.Errorf("token refresh transaction: %w", err)
	}

	return newAccessToken, newRefreshToken, nil
}
