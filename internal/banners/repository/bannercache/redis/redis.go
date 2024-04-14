package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/Leopold1975/banners_control/internal/banners/domain/models"
	"github.com/Leopold1975/banners_control/internal/banners/repository/bannerrepo"
	"github.com/Leopold1975/banners_control/internal/pkg/config"
	"github.com/Leopold1975/banners_control/internal/pkg/redistools"
	"github.com/redis/go-redis/v9"
)

type BannerCache struct {
	rdb     *redis.Client
	expTime time.Duration
}

func New(ctx context.Context, cfg config.RedisCache) (BannerCache, error) {
	rdb := redis.NewClient(&redis.Options{ //nolint:exhaustruct
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := redistools.Connect(ctx, rdb); err != nil {
		return BannerCache{}, fmt.Errorf("connect error: %w", err)
	}

	return BannerCache{
		rdb:     rdb,
		expTime: cfg.ExpTime,
	}, nil
}

func (bc BannerCache) CreateBanner(ctx context.Context, banner models.Banner) error {
	bannerJSON, err := json.Marshal(banner)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	_, err = bc.rdb.Set(ctx, fmt.Sprintf("banner:%d", banner.ID), bannerJSON, bc.expTime).Result() //nolint:perfsprint
	if err != nil {
		return fmt.Errorf("set error: %w", err)
	}

	// Создаем "индекс" для ускорения поиска по фиче и тэгу, т.к.
	// фича и тэг "required" для пользователя.
	for _, tagID := range banner.Tags {
		_, err = bc.rdb.SAdd(ctx, fmt.Sprintf("feature:%d:tag:%d", banner.FeatureID, tagID), banner.ID).Result()
		if err != nil {
			return fmt.Errorf("sadd error: %w", err)
		}
	}

	return nil
}

func (bc BannerCache) GetUserBanner(ctx context.Context, featureID, tagID int) (models.Banner, error) {
	banners, err := bc.rdb.SMembers(ctx, fmt.Sprintf("feature:%d:tag:%d", featureID, tagID)).Result()
	if err != nil {
		return models.Banner{}, fmt.Errorf("smembers error: %w", err)
	}

	for i := 0; i < len(banners); i++ {
		j := rand.Intn(len(banners)) //nolint:gosec

		bannerJSON, err := bc.rdb.Get(ctx, fmt.Sprintf("banner:%s", banners[j])).Result() //nolint:perfsprint
		if errors.Is(err, redis.Nil) {
			continue
		} else if err != nil {
			return models.Banner{}, fmt.Errorf("get error: %w", err)
		}

		var banner models.Banner

		err = json.Unmarshal([]byte(bannerJSON), &banner)
		if err != nil {
			return models.Banner{}, fmt.Errorf("unmarshal error: %w", err)
		}

		if banner.Active {
			return banner, nil
		}
	}

	return models.Banner{}, bannerrepo.ErrNotFound
}

func (bc BannerCache) DeleteBanner(ctx context.Context, bannerID int) error {
	key := fmt.Sprintf("banner:%d", bannerID)

	deleted, err := bc.rdb.Del(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("del error: %w", err)
	}

	if deleted == 0 {
		return bannerrepo.ErrNotFound
	}

	return nil
}
