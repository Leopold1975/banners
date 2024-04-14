package authservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/Leopold1975/banners_control/internal/banners/domain/models"
	"github.com/Leopold1975/banners_control/internal/pkg/config"
	"github.com/Leopold1975/banners_control/internal/pkg/jwtauth"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo Repository
	cfg      config.Auth
}

const (
	adminRole = "admin"
)

var ErrNotAllowed = errors.New("only admins can create admin")

type Repository interface {
	CreateUser(context.Context, models.User) error
	GetUser(context.Context, string) (models.User, error)
}

func New(userRepo Repository, cfg config.Auth) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

func (as *AuthService) CreateUser(ctx context.Context, req CreateUserRequest) (string, error) {
	if req.Role == adminRole { // только админы могут создавать админов
		isAdmin, err := as.Auth(req.Token)
		if err != nil {
			return "", fmt.Errorf("auth error: %w", err)
		}

		if !isAdmin {
			return "", ErrNotAllowed
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("generate from password error: %w", err)
	}

	var u models.User

	u.Username = req.Username
	u.PasswordHash = string(hash)
	u.Role = req.Role
	u.Tags = req.Tags
	u.Feature = req.Feature

	err = as.userRepo.CreateUser(ctx, u)
	if err != nil {
		return "", fmt.Errorf("create user error: %w", err)
	}

	token, err := jwtauth.GetToken(u, as.cfg.TTL, as.cfg.Secret)
	if err != nil {
		return "", fmt.Errorf("can't get token error: %w", err)
	}

	return token, nil
}

func (as *AuthService) Auth(token string) (bool, error) {
	role, err := jwtauth.ValidateTokenRole(token, as.cfg.Secret)
	if err != nil {
		return false, fmt.Errorf("validate token role error: %w", err)
	}

	return role == adminRole, nil
}

func (as *AuthService) Login(ctx context.Context, username, password string) (string, error) {
	u, err := as.userRepo.GetUser(ctx, username)
	if err != nil {
		return "", fmt.Errorf("get user error: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	if err != nil {
		return "", fmt.Errorf("compare password error: %w", err)
	}

	token, err := jwtauth.GetToken(u, as.cfg.TTL, as.cfg.Secret)
	if err != nil {
		return "", fmt.Errorf("can't get token error: %w", err)
	}

	return token, nil
}
