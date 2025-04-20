package entity

import "github.com/golang-jwt/jwt/v5"

type TokenClaims struct {
	UserID   string `json:"user_id"`
	ClientIP string `json:"client_ip"`
	jwt.RegisteredClaims
}
