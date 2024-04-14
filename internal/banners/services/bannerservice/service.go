package bannerservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Leopold1975/banners_control/internal/banners/domain/models"
	repo "github.com/Leopold1975/banners_control/internal/banners/repository/bannerrepo"
	"github.com/Leopold1975/banners_control/pkg/logger"
)

type BannerService struct {
	bannerRepo  Repository
	bannerCache Cache
	lg          logger.Logger
}

type Repository interface {
	CreateBanner(context.Context, models.Banner) (int, error)
	UpdateBanner(context.Context, models.Banner) error
	DeleteBanner(context.Context, int) error
	GetBannerByFeatureAndTags(context.Context, repo.GetBannerRequest) ([]models.Banner, error)
	Shutdown(context.Context) error
}

type Cache interface {
	GetUserBanner(ctx context.Context, featureID int, tagID int) (models.Banner, error)
	CreateBanner(context.Context, models.Banner) error
	DeleteBanner(context.Context, int) error
}

func New(bannerRepo Repository, bannerCache Cache, lg logger.Logger) *BannerService {
	return &BannerService{
		bannerRepo:  bannerRepo,
		bannerCache: bannerCache,
		lg:          lg,
	}
}

func (bs *BannerService) GetBanner(ctx context.Context, req GetBannerRequest) ([]models.Banner, error) {
	repoReq := repo.GetBannerRequest{
		FeatureID:  req.FeatureID,
		Tags:       req.Tags,
		Offset:     req.Offset,
		Limit:      req.Limit,
		OnlyActive: !req.IsAdmin,
	}

	if !req.UseLastRevision && !req.IsAdmin {
		b, err := bs.bannerCache.GetUserBanner(ctx, repoReq.FeatureID, repoReq.Tags[0])
		if err != nil {
			bs.lg.Info("cache missed")
			bs.lg.Error("delete banner cache error: %s", err.Error())
		} else {
			bs.lg.Info("cache hit")

			return []models.Banner{b}, nil
		}
	}

	banners, err := bs.bannerRepo.GetBannerByFeatureAndTags(ctx, repoReq)
	if err != nil {
		return nil, fmt.Errorf("get banner error: %w", err)
	}

	return banners, nil
}

func (bs *BannerService) CreateBanner(ctx context.Context, b models.Banner) (int, error) {
	b.CreatedAt = time.Now()
	b.UpdatedAt = time.Now()

	id, err := bs.bannerRepo.CreateBanner(ctx, b)
	if err != nil {
		return 0, fmt.Errorf("create banner error: %w", err)
	}

	b.ID = int64(id)

	if err := bs.bannerCache.CreateBanner(ctx, b); err != nil {
		bs.lg.Error("create banner cache error: %s", err.Error)
	}

	return id, nil
}

func (bs BannerService) DeleteBanner(ctx context.Context, id int) error {
	if err := bs.bannerCache.DeleteBanner(ctx, id); err != nil {
		bs.lg.Error("delete banner cache errorL %s", err.Error())
	}

	if err := bs.bannerRepo.DeleteBanner(ctx, id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		}
	}

	return nil
}

func (bs *BannerService) UpdateBanner(ctx context.Context, banner models.Banner) error {
	if err := bs.bannerRepo.UpdateBanner(ctx, banner); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		}

		return fmt.Errorf("update banner error: %w", err)
	}

	return nil
}

func (bs *BannerService) BackroundRefresh(ctx context.Context, ttl time.Duration) {
	t := time.NewTicker(ttl)

	err := bs.refresh(ctx)
	if err != nil {
		bs.lg.Error("refresh error: %s", err.Error())
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			err = bs.refresh(ctx)
			if err != nil {
				bs.lg.Error("refresh error: %s", err.Error())
			}
		}
	}
}

func (bs *BannerService) Shutdown(ctx context.Context) error {
	if err := bs.bannerRepo.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown banner repo error: %w", err)
	}

	return nil
}

func (bs *BannerService) refresh(ctx context.Context) error {
	errCh := make(chan error)
	go func() {
		defer close(errCh)

		var req repo.GetBannerRequest
		req.FeatureID = -1

		banners, err := bs.bannerRepo.GetBannerByFeatureAndTags(ctx, req)
		if err != nil {
			errCh <- fmt.Errorf("get banners error: %w", err)

			return
		}

		for _, b := range banners {
			if err := bs.bannerCache.CreateBanner(ctx, b); err != nil {
				errCh <- fmt.Errorf("create banner cache error: %w", err)

				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled error: %w", ctx.Err())
	case err := <-errCh:
		if err != nil {
			bs.lg.Error("refresh error: %s", err.Error())

			return err
		}

		return nil
	}
}
