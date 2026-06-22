package main

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct{ secret []byte }

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type PINTokenClaims struct {
	UserID   string `json:"user_id"`
	MemoryID string `json:"memory_id"`
	jwt.RegisteredClaims
}

func NewJWTService(secret string) *JWTService {
	if secret == "" {
		secret = "default-secret-change-me"
	}
	return &JWTService{secret: []byte(secret)}
}

func (s *JWTService) GenerateToken(userID, email string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

func (s *JWTService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func (s *JWTService) GeneratePINToken(userID, memoryID string) (string, error) {
	claims := PINTokenClaims{
		UserID:   userID,
		MemoryID: memoryID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

func (s *JWTService) ValidatePINToken(tokenStr, memoryID string) (*PINTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &PINTokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*PINTokenClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid PIN token")
	}
	if claims.MemoryID != memoryID {
		return nil, errors.New("PIN token not valid for this memory")
	}
	return claims, nil
}