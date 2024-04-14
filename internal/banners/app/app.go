package app

import (
	"context"
	"fmt"
	"time"

	"github.com/Leopold1975/banners_control/internal/banners/api/server"
	"github.com/Leopold1975/banners_control/internal/banners/repository/bannercache/redis"
	br "github.com/Leopold1975/banners_control/internal/banners/repository/bannerrepo/postgres"
	ur "github.com/Leopold1975/banners_control/internal/banners/repository/userrepo/postgres"
	"github.com/Leopold1975/banners_control/internal/banners/services/authservice"
	"github.com/Leopold1975/banners_control/internal/banners/services/bannerservice"
	"github.com/Leopold1975/banners_control/internal/pkg/config"
	"github.com/Leopold1975/banners_control/pkg/logger"
)

type Server interface {
	Start(context.Context) error
	Shutdown(context.Context) error
}

type BannersApp struct {
	s   Server
	lg  logger.Logger
	cfg config.Config
}

func New(ctx context.Context, cfg config.Config) (BannersApp, error) {
	lg, err := logger.New(cfg.Logger)
	if err != nil {
		return BannersApp{}, fmt.Errorf("can't get logger error: %w", err)
	}

	bannerRepo, err := br.New(ctx, cfg.PostgresDB)
	if err != nil {
		return BannersApp{}, fmt.Errorf("postgres banner repo initializing error: %w", err)
	}

	bc, err := redis.New(ctx, cfg.RedisCache)
	if err != nil {
		return BannersApp{}, fmt.Errorf("redis banner cache initializing error: %w", err)
	}

	bannerService := bannerservice.New(bannerRepo, bc, lg)

	go bannerService.BackroundRefresh(ctx, cfg.RedisCache.ExpTime)

	userRepo, err := ur.New(ctx, cfg.PostgresDB)
	if err != nil {
		return BannersApp{}, fmt.Errorf("postgres user repo initializing error: %w", err)
	}

	authService := authservice.New(userRepo, cfg.Auth)

	s := server.New(cfg.Server, bannerService, authService, lg)

	return BannersApp{
		s:   s,
		lg:  lg,
		cfg: cfg,
	}, nil
}

func (ba *BannersApp) Run(ctx context.Context) {
	ba.lg.Infof("STARTED SERVER ON %s", ba.cfg.Server.Addr)

	go func() {
		if err := ba.s.Start(ctx); err != nil {
			ba.lg.Errorf("server start error: %s", err.Error())
			ctx.Done()

			return
		}
	}()

	<-ctx.Done()

	ctxS, cancel := context.WithTimeout(context.Background(), time.Second*5) //nolint:gomnd
	defer cancel()

	if err := ba.Stop(ctxS); err != nil { //nolint:contextcheck
		ba.lg.Errorf("server shutdown error: %s", err.Error())
	}
}

func (ba *BannersApp) Stop(ctx context.Context) error {
	if err := ba.s.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	ba.lg.Info("Shutdowned successfully")

	return nil
}
