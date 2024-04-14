package jwtauth

import (
	"errors"
	"fmt"
	"time"

	"github.com/Leopold1975/banners_control/internal/banners/domain/models"
	"github.com/golang-jwt/jwt"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
	ErrNoClaim      = errors.New("required claim not found")
)

func GetToken(user models.User, ttl time.Duration, secret string) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", ErrInvalidToken
	}

	if user.Role == "" {
		return "", ErrNoClaim
	}

	claims["role"] = user.Role
	claims["exp"] = time.Now().Add(ttl).Unix()

	t, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign string error: %w", err)
	}

	return t, nil
}

func ValidateTokenRole(tokenString string, secret string) (string, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		return []byte(secret), nil
	})
	if err != nil {
		jwtErr := new(jwt.ValidationError)
		if errors.As(err, &jwtErr) {
			if jwtErr.Errors == jwt.ValidationErrorExpired {
				return "", ErrTokenExpired
			}
		}

		return "", fmt.Errorf("parse token error: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		role, ok := claims["role"].(string)
		if !ok {
			return "", ErrNoClaim
		}

		exp, ok := claims["exp"].(float64)
		if !ok {
			return "", ErrNoClaim
		}

		if int64(exp) < time.Now().Unix() {
			return "", ErrTokenExpired
		}

		return role, nil
	}

	return "", ErrInvalidToken
}
